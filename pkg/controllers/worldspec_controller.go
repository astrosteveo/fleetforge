/*
Copyright 2024 FleetForge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
	"github.com/astrosteveo/fleetforge/pkg/cell"
)

const (
	// ForceSplitAnnotation is the annotation key used to trigger manual cell splits
	ForceSplitAnnotation = "fleetforge.io/force-split"
)

// WorldSpecReconciler reconciles a WorldSpec object
type WorldSpecReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Log         logr.Logger
	Recorder    record.EventRecorder
	CellManager cell.CellManager
}

//+kubebuilder:rbac:groups=fleetforge.io,resources=worldspecs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=fleetforge.io,resources=worldspecs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=fleetforge.io,resources=worldspecs/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile reconciles the WorldSpec resource
func (r *WorldSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("worldspec", req.NamespacedName)

	// Fetch the WorldSpec instance
	worldSpec := &fleetforgev1.WorldSpec{}
	err := r.Get(ctx, req.NamespacedName, worldSpec)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("WorldSpec resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue with exponential backoff
		log.Error(err, "Failed to get WorldSpec")
		return r.requeueWithBackoff(err), err
	}

	log.Info("Reconciling WorldSpec", "phase", worldSpec.Status.Phase, "generation", worldSpec.Generation)

	// Set initial phase if not set
	if worldSpec.Status.Phase == "" {
		worldSpec.Status.Phase = "Initializing"
		worldSpec.Status.Message = "Starting world initialization"
		if err := r.updateStatus(ctx, worldSpec, log); err != nil {
			return r.requeueWithBackoff(err), err
		}
		// Emit event for world initialization start
		r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "InitializationStarted",
			fmt.Sprintf("World %s initialization started", worldSpec.Name))
	}

	// Check for manual split override annotation
	if forceSplitValue, hasAnnotation := worldSpec.Annotations[ForceSplitAnnotation]; hasAnnotation {
		if err := r.handleManualSplitOverride(ctx, worldSpec, forceSplitValue, log); err != nil {
			log.Error(err, "Failed to handle manual split override")
			r.Recorder.Event(worldSpec, corev1.EventTypeWarning, "ManualSplitFailed",
				fmt.Sprintf("Manual split override failed: %v", err))
		}
	}

	// Handle creation/update of cell pods
	result, err := r.reconcileCells(ctx, worldSpec, log)
	if err != nil {
		log.Error(err, "Failed to reconcile cells")
		r.handleReconcileError(ctx, worldSpec, err, log)
		return result, err
	}

	// Update the status based on current state
	if err := r.updateWorldSpecStatus(ctx, worldSpec, log); err != nil {
		log.Error(err, "Failed to update status")
		return r.requeueWithBackoff(err), err
	}

	// Requeue for status updates and annotation monitoring
	if _, hasAnnotation := worldSpec.Annotations[ForceSplitAnnotation]; hasAnnotation {
		// Aggressive polling if manual split annotation is present
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	// Default (less aggressive) polling interval
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// reconcileCells manages the lifecycle of cell pods
func (r *WorldSpecReconciler) reconcileCells(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) (ctrl.Result, error) {
	// Calculate cell boundaries based on world topology
	cells := calculateCellBoundaries(worldSpec.Spec.Topology)

	// Validate that cell boundaries properly partition the parent space
	// Use a small tolerance for floating-point precision issues
	tolerance := 1e-6
	if err := validateCellPartitioning(worldSpec.Spec.Topology.WorldBoundaries, cells, tolerance); err != nil {
		log.Error(err, "Cell boundary validation failed", "worldSpec", worldSpec.Name)
		r.Recorder.Event(worldSpec, corev1.EventTypeWarning, "ValidationFailed",
			fmt.Sprintf("Cell boundary validation failed: %v", err))
		return ctrl.Result{}, fmt.Errorf("cell boundary validation failed: %w", err)
	}

	log.Info("Cell boundary validation passed",
		"worldSpec", worldSpec.Name,
		"parentArea", worldSpec.Spec.Topology.WorldBoundaries.CalculateArea(),
		"numCells", len(cells))

	// Track cells being processed for status updates
	cellsCreated := 0
	cellsUpdated := 0
	cellsErrored := 0

	for i, cellBounds := range cells {
		cellID := fmt.Sprintf("%s-cell-%d", worldSpec.Name, i)

		// Create or update deployment for this cell
		deploymentResult, err := r.reconcileCellDeployment(ctx, worldSpec, cellID, cellBounds, log)
		if err != nil {
			cellsErrored++
			log.Error(err, "Failed to reconcile cell deployment", "cellID", cellID)
			r.Recorder.Event(worldSpec, corev1.EventTypeWarning, "CellDeploymentFailed",
				fmt.Sprintf("Failed to reconcile deployment for cell %s: %v", cellID, err))
			return ctrl.Result{}, fmt.Errorf("failed to reconcile cell deployment %s: %w", cellID, err)
		}

		if deploymentResult == controllerutil.OperationResultCreated {
			cellsCreated++
			r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "CellDeploymentCreated",
				fmt.Sprintf("Created deployment for cell %s", cellID))
		} else if deploymentResult == controllerutil.OperationResultUpdated {
			cellsUpdated++
			r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "CellDeploymentUpdated",
				fmt.Sprintf("Updated deployment for cell %s", cellID))
		}

		// Create or update service for this cell
		serviceResult, err := r.reconcileCellService(ctx, worldSpec, cellID, log)
		if err != nil {
			cellsErrored++
			log.Error(err, "Failed to reconcile cell service", "cellID", cellID)
			r.Recorder.Event(worldSpec, corev1.EventTypeWarning, "CellServiceFailed",
				fmt.Sprintf("Failed to reconcile service for cell %s: %v", cellID, err))
			return ctrl.Result{}, fmt.Errorf("failed to reconcile cell service %s: %w", cellID, err)
		}

		if serviceResult == controllerutil.OperationResultCreated {
			r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "CellServiceCreated",
				fmt.Sprintf("Created service for cell %s", cellID))
		} else if serviceResult == controllerutil.OperationResultUpdated {
			r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "CellServiceUpdated",
				fmt.Sprintf("Updated service for cell %s", cellID))
		}
	}

	// Log summary of cell reconciliation
	log.Info("Cell reconciliation completed",
		"cellsCreated", cellsCreated,
		"cellsUpdated", cellsUpdated,
		"cellsErrored", cellsErrored,
		"totalCells", len(cells))

	// Emit summary events if there were significant changes
	if cellsCreated > 0 {
		r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "CellsCreated",
			fmt.Sprintf("Created %d new cell deployments for world %s", cellsCreated, worldSpec.Name))
	}

	if cellsUpdated > 0 {
		r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "CellsUpdated",
			fmt.Sprintf("Updated %d cell deployments for world %s", cellsUpdated, worldSpec.Name))
	}

	return ctrl.Result{}, nil
}

// reconcileCellDeployment creates or updates a deployment for a cell
func (r *WorldSpecReconciler) reconcileCellDeployment(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, cellID string, bounds fleetforgev1.WorldBounds, log logr.Logger) (controllerutil.OperationResult, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cellID,
			Namespace: worldSpec.Namespace,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(worldSpec, deployment, r.Scheme); err != nil {
			return err
		}

		// Configure deployment spec
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     "fleetforge-cell",
					"cell-id": cellID,
					"world":   worldSpec.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":     "fleetforge-cell",
						"cell-id": cellID,
						"world":   worldSpec.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cell-simulator",
							Image: worldSpec.Spec.GameServerImage,
							Args: []string{
								fmt.Sprintf("--cell-id=%s", cellID),
								fmt.Sprintf("--x-min=%f", bounds.XMin),
								fmt.Sprintf("--x-max=%f", bounds.XMax),
								fmt.Sprintf("--max-players=%d", worldSpec.Spec.Capacity.MaxPlayersPerCell),
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    r.parseResourceQuantity(worldSpec.Spec.Capacity.CPULimitPerCell, "cpu"),
									corev1.ResourceMemory: r.parseResourceQuantity(worldSpec.Spec.Capacity.MemoryLimitPerCell, "memory"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    r.parseResourceQuantity(getStringValue(worldSpec.Spec.Capacity.CPURequestPerCell, "500m"), "cpu"),
									corev1.ResourceMemory: r.parseResourceQuantity(getStringValue(worldSpec.Spec.Capacity.MemoryRequestPerCell, "1Gi"), "memory"),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "health",
									ContainerPort: 8081,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromString("health"),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromString("health"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
		}

		return nil
	})

	if err != nil {
		log.Error(err, "Failed to create or update deployment", "deployment", cellID)
		return controllerutil.OperationResultNone, err
	}

	log.Info("Successfully reconciled deployment", "deployment", cellID, "operation", result)
	return result, nil
}

// reconcileCellService creates or updates a service for a cell
func (r *WorldSpecReconciler) reconcileCellService(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, cellID string, log logr.Logger) (controllerutil.OperationResult, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cellID + "-service",
			Namespace: worldSpec.Namespace,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(worldSpec, service, r.Scheme); err != nil {
			return err
		}

		// Configure service spec
		service.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"app":     "fleetforge-cell",
				"cell-id": cellID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "health",
					Port:       8081,
					TargetPort: intstr.FromString("health"),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       8080,
					TargetPort: intstr.FromString("metrics"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		}

		return nil
	})

	if err != nil {
		log.Error(err, "Failed to create or update service", "service", cellID+"-service")
		return controllerutil.OperationResultNone, err
	}

	log.Info("Successfully reconciled service", "service", cellID+"-service", "operation", result)
	return result, nil
}

// UpdateWorldSpecStatus updates the status of the WorldSpec (exported for testing)
func (r *WorldSpecReconciler) UpdateWorldSpecStatus(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) error {
	return r.updateWorldSpecStatus(ctx, worldSpec, log)
}

// updateWorldSpecStatus updates the status of the WorldSpec
func (r *WorldSpecReconciler) updateWorldSpecStatus(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) error {
	// Count active cells by looking at deployments
	deploymentList := &appsv1.DeploymentList{}
	err := r.List(ctx, deploymentList, client.InNamespace(worldSpec.Namespace), client.MatchingLabels{"world": worldSpec.Name})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	// Also get pod information for more detailed status
	podList := &corev1.PodList{}
	err = r.List(ctx, podList, client.InNamespace(worldSpec.Namespace), client.MatchingLabels{"world": worldSpec.Name})
	if err != nil {
		log.Error(err, "Failed to list pods, continuing with deployment-only status")
	}

	// Create a map of cell ID to pod for quick lookup
	podMap := make(map[string]*corev1.Pod)
	for i := range podList.Items {
		pod := &podList.Items[i]
		if cellID, ok := pod.Labels["cell-id"]; ok {
			podMap[cellID] = pod
		}
	}

	activeCells := int32(0)
	expectedCells := worldSpec.Spec.Topology.InitialCells
	var cellStatuses []fleetforgev1.CellStatus
	totalPlayers := int32(0) // This would be populated from actual metrics in a real implementation

	for _, deployment := range deploymentList.Items {
		cellID := deployment.Name
		cellStatus := fleetforgev1.CellStatus{
			ID:         cellID,
			PodName:    "",
			Health:     "Unknown",
			Boundaries: r.getCellBoundaries(cellID, worldSpec), // Helper to get boundaries from deployment
		}

		// Check if there's a corresponding pod
		if pod, exists := podMap[cellID]; exists {
			cellStatus.PodName = pod.Name
			cellStatus.Health = r.getPodHealthStatus(pod)

			// Update last heartbeat from pod condition
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady {
					cellStatus.LastHeartbeat = &condition.LastTransitionTime
					break
				}
			}
		}

		// Determine if cell is active based on deployment and pod status
		if deployment.Status.ReadyReplicas > 0 && cellStatus.Health == "Healthy" {
			activeCells++
		} else if cellStatus.Health == "Unknown" && deployment.Status.ReadyReplicas > 0 {
			// Deployment is ready but pod status unknown - consider it active but with caution
			activeCells++
			cellStatus.Health = "Pending"
		} else if deployment.Status.ReadyReplicas == 0 {
			cellStatus.Health = "Pending"
		}

		cellStatuses = append(cellStatuses, cellStatus)
	}

	// Update basic status fields
	worldSpec.Status.ActiveCells = activeCells
	worldSpec.Status.TotalPlayers = totalPlayers // Would be updated from metrics
	worldSpec.Status.Cells = cellStatuses
	worldSpec.Status.LastUpdateTime = &metav1.Time{Time: time.Now()}

	// Determine if world is ready (all expected cells are active and healthy)
	isReady := activeCells >= expectedCells && expectedCells > 0
	wasReady := r.isWorldReady(worldSpec)

	// Update phase based on readiness and cell health
	if isReady {
		// Check if all cells are actually healthy, not just ready
		allHealthy := true
		for _, cell := range cellStatuses {
			if cell.Health != "Healthy" {
				allHealthy = false
				break
			}
		}

		if allHealthy {
			worldSpec.Status.Phase = "Running"
			worldSpec.Status.Message = fmt.Sprintf("World running with %d/%d healthy cells", activeCells, expectedCells)
		} else {
			// Some cells are pending/starting but ready - still consider running for backwards compatibility
			worldSpec.Status.Phase = "Running"
			worldSpec.Status.Message = fmt.Sprintf("World running with %d/%d cells ready", activeCells, expectedCells)
		}
	} else {
		// Determine if we're still creating or if there's an issue
		if len(deploymentList.Items) < int(expectedCells) {
			worldSpec.Status.Phase = "Creating"
			worldSpec.Status.Message = fmt.Sprintf("Creating cell deployments (%d/%d)", len(deploymentList.Items), expectedCells)
		} else {
			worldSpec.Status.Phase = "Initializing"
			worldSpec.Status.Message = fmt.Sprintf("Waiting for cells to become ready (%d/%d)", activeCells, expectedCells)
		}
	}

	// Update Ready condition
	readyCondition := metav1.Condition{
		Type:               "Ready",
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: worldSpec.Generation,
	}

	if isReady {
		readyCondition.Status = metav1.ConditionTrue
		readyCondition.Reason = "AllCellsReady"
		readyCondition.Message = fmt.Sprintf("All %d expected cells are ready and running", expectedCells)
	} else {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Reason = "CellsNotReady"
		readyCondition.Message = fmt.Sprintf("Waiting for cells to become ready (%d/%d)", activeCells, expectedCells)
	}

	// Update the condition in the status
	meta.SetStatusCondition(&worldSpec.Status.Conditions, readyCondition)

	// Fire WorldInitialized event when transitioning to ready for the first time
	if isReady && !wasReady {
		r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "WorldInitialized",
			fmt.Sprintf("World %s initialized with %d cells", worldSpec.Name, activeCells))
		log.Info("World initialized", "world", worldSpec.Name, "activeCells", activeCells)
	}

	// Fire additional lifecycle events
	if !isReady && wasReady {
		r.Recorder.Event(worldSpec, corev1.EventTypeWarning, "WorldDegraded",
			fmt.Sprintf("World %s degraded - %d/%d cells ready", worldSpec.Name, activeCells, expectedCells))
		log.Info("World degraded", "world", worldSpec.Name, "activeCells", activeCells, "expectedCells", expectedCells)
	}

	return r.updateStatus(ctx, worldSpec, log)
}

// isWorldReady checks if the world was previously in Ready condition
func (r *WorldSpecReconciler) isWorldReady(worldSpec *fleetforgev1.WorldSpec) bool {
	readyCondition := meta.FindStatusCondition(worldSpec.Status.Conditions, "Ready")
	return readyCondition != nil && readyCondition.Status == metav1.ConditionTrue
}

// updateStatus updates the status subresource
func (r *WorldSpecReconciler) updateStatus(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) error {
	if err := r.Status().Update(ctx, worldSpec); err != nil {
		log.Error(err, "Failed to update WorldSpec status")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorldSpecReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&fleetforgev1.WorldSpec{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// Helper functions

// validateCellPartitioning verifies that child cells properly partition the parent space
// without overlaps or gaps, and that the sum of areas equals the parent area
func validateCellPartitioning(parentBounds fleetforgev1.WorldBounds, childCells []fleetforgev1.WorldBounds, tolerance float64) error {
	if len(childCells) == 0 {
		return fmt.Errorf("no child cells provided for validation")
	}

	if !parentBounds.IsValidBounds() {
		return fmt.Errorf("parent bounds are invalid")
	}

	// Calculate parent area
	parentArea := parentBounds.CalculateArea()
	if parentArea == 0 {
		return fmt.Errorf("parent area is zero")
	}

	// Calculate sum of child areas and validate each child
	var childAreaSum float64
	for i, child := range childCells {
		if !child.IsValidBounds() {
			return fmt.Errorf("child cell %d has invalid bounds", i)
		}

		childArea := child.CalculateArea()
		if childArea == 0 {
			return fmt.Errorf("child cell %d has zero area", i)
		}

		childAreaSum += childArea

		// Validate child is within parent bounds
		if err := validateChildWithinParent(parentBounds, child); err != nil {
			return fmt.Errorf("child cell %d not within parent bounds: %w", i, err)
		}
	}

	// Check if areas match within tolerance
	areaDifference := childAreaSum - parentArea
	if areaDifference < 0 {
		areaDifference = -areaDifference
	}

	if areaDifference > tolerance {
		return fmt.Errorf("area mismatch: child areas sum to %f, parent area is %f, difference %f exceeds tolerance %f",
			childAreaSum, parentArea, areaDifference, tolerance)
	}

	// Validate no overlaps between child cells
	if err := validateNoOverlaps(childCells); err != nil {
		return fmt.Errorf("overlap detected between child cells: %w", err)
	}

	return nil
}

// validateChildWithinParent checks if a child bounds is completely within parent bounds
func validateChildWithinParent(parent, child fleetforgev1.WorldBounds) error {
	// Check X dimension
	if child.XMin < parent.XMin || child.XMax > parent.XMax {
		return fmt.Errorf("child X bounds [%f, %f] not within parent X bounds [%f, %f]",
			child.XMin, child.XMax, parent.XMin, parent.XMax)
	}

	// Check Y dimension if both have it
	if parent.YMin != nil && parent.YMax != nil && child.YMin != nil && child.YMax != nil {
		if *child.YMin < *parent.YMin || *child.YMax > *parent.YMax {
			return fmt.Errorf("child Y bounds [%f, %f] not within parent Y bounds [%f, %f]",
				*child.YMin, *child.YMax, *parent.YMin, *parent.YMax)
		}
	}

	// Check Z dimension if both have it
	if parent.ZMin != nil && parent.ZMax != nil && child.ZMin != nil && child.ZMax != nil {
		if *child.ZMin < *parent.ZMin || *child.ZMax > *parent.ZMax {
			return fmt.Errorf("child Z bounds [%f, %f] not within parent Z bounds [%f, %f]",
				*child.ZMin, *child.ZMax, *parent.ZMin, *parent.ZMax)
		}
	}

	return nil
}

// validateNoOverlaps checks that no two child cells overlap
func validateNoOverlaps(childCells []fleetforgev1.WorldBounds) error {
	for i := 0; i < len(childCells); i++ {
		for j := i + 1; j < len(childCells); j++ {
			if boundsOverlap(childCells[i], childCells[j]) {
				return fmt.Errorf("child cells %d and %d overlap", i, j)
			}
		}
	}
	return nil
}

// boundsOverlap checks if two WorldBounds overlap
func boundsOverlap(bounds1, bounds2 fleetforgev1.WorldBounds) bool {
	// Check X dimension - no overlap if one is completely before the other
	if bounds1.XMax <= bounds2.XMin || bounds2.XMax <= bounds1.XMin {
		return false
	}

	// Check Y dimension if both have it
	if bounds1.YMin != nil && bounds1.YMax != nil && bounds2.YMin != nil && bounds2.YMax != nil {
		if *bounds1.YMax <= *bounds2.YMin || *bounds2.YMax <= *bounds1.YMin {
			return false
		}
	}

	// Check Z dimension if both have it
	if bounds1.ZMin != nil && bounds1.ZMax != nil && bounds2.ZMin != nil && bounds2.ZMax != nil {
		if *bounds1.ZMax <= *bounds2.ZMin || *bounds2.ZMax <= *bounds1.ZMin {
			return false
		}
	}

	// If we get here, there is overlap in all checked dimensions
	return true
}

// CalculateCellBoundaries calculates cell boundaries based on topology (exported for testing)
func CalculateCellBoundaries(topology fleetforgev1.WorldTopology) []fleetforgev1.WorldBounds {
	return calculateCellBoundaries(topology)
}

func calculateCellBoundaries(topology fleetforgev1.WorldTopology) []fleetforgev1.WorldBounds {
	// Simple implementation: divide world into equal cells
	cells := make([]fleetforgev1.WorldBounds, topology.InitialCells)

	worldWidth := topology.WorldBoundaries.XMax - topology.WorldBoundaries.XMin
	cellWidth := worldWidth / float64(topology.InitialCells)

	for i := int32(0); i < topology.InitialCells; i++ {
		xMin := topology.WorldBoundaries.XMin + (float64(i) * cellWidth)
		xMax := xMin + cellWidth

		cells[i] = fleetforgev1.WorldBounds{
			XMin: xMin,
			XMax: xMax,
			YMin: topology.WorldBoundaries.YMin,
			YMax: topology.WorldBoundaries.YMax,
			ZMin: topology.WorldBoundaries.ZMin,
			ZMax: topology.WorldBoundaries.ZMax,
		}
	}

	return cells
}

func int32Ptr(i int32) *int32 {
	return &i
}

func getStringValue(ptr *string, defaultValue string) string {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// handleManualSplitOverride processes manual split override annotations
func (r *WorldSpecReconciler) handleManualSplitOverride(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, cellIDSpec string, log logr.Logger) error {
	// Initialize cell manager if not already done
	if r.CellManager == nil {
		r.CellManager = cell.NewCellManager()
	}

	// Parse the annotation value - could be a specific cell ID or "all"
	cellIDs := r.parseCellIDsFromAnnotation(cellIDSpec, worldSpec)

	if len(cellIDs) == 0 {
		return fmt.Errorf("no valid cell IDs found in annotation value: %s", cellIDSpec)
	}

	// Extract user identity from object metadata
	userInfo := r.extractUserIdentity(worldSpec)

	var splitErrors []string
	successfulSplits := 0

	for _, cellID := range cellIDs {
		log.Info("Processing manual split override", "cellID", cellID, "userInfo", userInfo)

		// Use CellManager to perform the manual split
		childCells, err := r.CellManager.ManualSplitCell(cell.CellID(cellID), userInfo)
		if err != nil {
			splitErrors = append(splitErrors, fmt.Sprintf("cell %s: %v", cellID, err))
			log.Error(err, "Manual split failed", "cellID", cellID)
			continue
		}

		successfulSplits++
		log.Info("Manual split successful", "cellID", cellID, "childCells", len(childCells))

		// Record event for successful manual split
		r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "ManualOverride",
			fmt.Sprintf("Cell %s manually split into %d children by user", cellID, len(childCells)))
	}

	// Remove the annotation after processing to prevent re-processing
	if err := r.removeAnnotation(ctx, worldSpec, log); err != nil {
		log.Error(err, "Failed to remove manual split annotation")
		return err
	}

	// Record summary event
	if successfulSplits > 0 {
		r.Recorder.Event(worldSpec, corev1.EventTypeNormal, "ManualOverride",
			fmt.Sprintf("Manual split override completed: %d successful, %d failed", successfulSplits, len(splitErrors)))
	}

	if len(splitErrors) > 0 {
		return fmt.Errorf("manual split failures: %s", strings.Join(splitErrors, "; "))
	}

	return nil
}

// parseCellIDsFromAnnotation parses the annotation value to extract cell IDs
func (r *WorldSpecReconciler) parseCellIDsFromAnnotation(value string, worldSpec *fleetforgev1.WorldSpec) []string {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	// Handle "all" keyword to split all active cells
	if strings.ToLower(value) == "all" {
		var cellIDs []string
		for i := int32(0); i < worldSpec.Spec.Topology.InitialCells; i++ {
			cellID := fmt.Sprintf("%s-cell-%d", worldSpec.Name, i)
			cellIDs = append(cellIDs, cellID)
		}
		return cellIDs
	}

	// Handle comma-separated cell IDs
	cellIDs := strings.Split(value, ",")
	var result []string
	for _, cellID := range cellIDs {
		cellID = strings.TrimSpace(cellID)
		if cellID != "" {
			result = append(result, cellID)
		}
	}

	return result
}

// extractUserIdentity extracts user identity information from the object metadata
func (r *WorldSpecReconciler) extractUserIdentity(worldSpec *fleetforgev1.WorldSpec) map[string]interface{} {
	userInfo := make(map[string]interface{})

	// Extract user information from annotations and managed fields
	if worldSpec.Annotations != nil {
		if user, exists := worldSpec.Annotations["kubectl.kubernetes.io/last-applied-by"]; exists {
			userInfo["applied_by"] = user
		}
	}

	// Extract user from managed fields (Kubernetes tracks who made changes)
	if len(worldSpec.ManagedFields) > 0 {
		latestField := worldSpec.ManagedFields[len(worldSpec.ManagedFields)-1]
		if latestField.Manager != "" {
			userInfo["manager"] = latestField.Manager
		}
		if latestField.Time != nil {
			userInfo["timestamp"] = latestField.Time.Time
		}
	}

	// Add world spec information for audit context
	userInfo["resource"] = fmt.Sprintf("WorldSpec/%s", worldSpec.Name)
	userInfo["namespace"] = worldSpec.Namespace
	userInfo["action"] = "manual_split_override"

	return userInfo
}

// removeAnnotation removes the force split annotation after processing
func (r *WorldSpecReconciler) removeAnnotation(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) error {
	if worldSpec.Annotations == nil {
		return nil
	}

	// Create a deep copy and remove the annotation from the copy
	patch := worldSpec.DeepCopy()
	delete(patch.Annotations, ForceSplitAnnotation)

	// Use MergeFrom to patch only the annotation field
	if err := r.Patch(ctx, patch, client.MergeFrom(worldSpec)); err != nil {
		return fmt.Errorf("failed to remove annotation via patch: %w", err)
	}

	log.Info("Removed manual split annotation", "annotation", ForceSplitAnnotation)
	return nil
}

// parseResourceQuantity parses a resource quantity string, logs errors, and uses contextually appropriate defaults.
func (r *WorldSpecReconciler) parseResourceQuantity(s string, resourceType string) resource.Quantity {
	qty, err := resource.ParseQuantity(s)
	if err != nil {
		var defaultVal string
		switch resourceType {
		case "cpu":
			defaultVal = "100m"
		case "memory":
			defaultVal = "128Mi"
		default:
			defaultVal = "100m"
		}
		r.Log.Error(err, "Failed to parse resource quantity", "value", s, "resourceType", resourceType, "usingDefault", defaultVal)
		return resource.MustParse(defaultVal)
	}
	return qty
}

// requeueWithBackoff implements exponential backoff for retries
func (r *WorldSpecReconciler) requeueWithBackoff(err error) ctrl.Result {
	// Base requeue time of 30 seconds with jitter
	baseDelay := time.Second * 30
	jitter := time.Duration(randInt(1, 10)) * time.Second

	if errors.IsConflict(err) {
		// Conflict errors should be retried quickly
		return ctrl.Result{RequeueAfter: time.Second * 5}
	} else if errors.IsServerTimeout(err) || errors.IsServiceUnavailable(err) {
		// Server issues should use longer backoff
		return ctrl.Result{RequeueAfter: baseDelay + jitter}
	} else if errors.IsInvalid(err) || errors.IsBadRequest(err) {
		// Invalid requests should be retried less frequently
		return ctrl.Result{RequeueAfter: time.Minute * 2}
	}

	// Default backoff
	return ctrl.Result{RequeueAfter: baseDelay}
}

// getRequeueInterval returns appropriate requeue interval based on world phase
func (r *WorldSpecReconciler) getRequeueInterval(worldSpec *fleetforgev1.WorldSpec) time.Duration {
	switch worldSpec.Status.Phase {
	case "Initializing", "Creating":
		// Check more frequently during initialization
		return time.Second * 30
	case "Running":
		// Less frequent checks when stable
		return time.Minute * 2
	case "Scaling":
		// More frequent during scaling operations
		return time.Second * 15
	case "Error":
		// Longer interval for error states to avoid thrashing
		return time.Minute * 5
	default:
		return time.Minute * 1
	}
}

// handleReconcileError handles errors during reconciliation with proper status and event updates
func (r *WorldSpecReconciler) handleReconcileError(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, err error, log logr.Logger) {
	// Categorize error type for better status reporting
	errorCategory := r.categorizeError(err)

	// Update status based on error type
	worldSpec.Status.Phase = "Error"
	worldSpec.Status.Message = fmt.Sprintf("%s: %s", errorCategory, err.Error())

	// Emit appropriate events based on error type
	eventType := corev1.EventTypeWarning
	eventReason := "ReconciliationError"

	if errors.IsNotFound(err) {
		eventReason = "ResourceNotFound"
	} else if errors.IsConflict(err) {
		eventReason = "ResourceConflict"
		eventType = corev1.EventTypeNormal // Conflicts are normal during concurrent updates
	} else if errors.IsInvalid(err) {
		eventReason = "InvalidConfiguration"
	} else if errors.IsServerTimeout(err) {
		eventReason = "ServerTimeout"
	}

	r.Recorder.Event(worldSpec, eventType, eventReason, worldSpec.Status.Message)

	// Update status
	if statusErr := r.updateStatus(ctx, worldSpec, log); statusErr != nil {
		log.Error(statusErr, "Failed to update status after reconcile error")
	}

	log.Error(err, "Reconciliation failed", "category", errorCategory, "reason", eventReason)
}

// categorizeError categorizes errors for better reporting
func (r *WorldSpecReconciler) categorizeError(err error) string {
	if errors.IsNotFound(err) {
		return "Resource Not Found"
	} else if errors.IsAlreadyExists(err) {
		return "Resource Already Exists"
	} else if errors.IsConflict(err) {
		return "Resource Conflict"
	} else if errors.IsInvalid(err) {
		return "Invalid Configuration"
	} else if errors.IsForbidden(err) {
		return "Insufficient Permissions"
	} else if errors.IsServerTimeout(err) {
		return "Server Timeout"
	} else if errors.IsServiceUnavailable(err) {
		return "Service Unavailable"
	} else if errors.IsInternalError(err) {
		return "Internal Server Error"
	} else {
		return "Unknown Error"
	}
}

// randInt returns a random integer between min and max (inclusive)
func randInt(min, max int) int {
	return min + int(time.Now().UnixNano())%(max-min+1)
}

// getCellBoundaries extracts cell boundaries from cell ID and world spec
func (r *WorldSpecReconciler) getCellBoundaries(cellID string, worldSpec *fleetforgev1.WorldSpec) fleetforgev1.WorldBounds {
	// Extract cell index from cell ID (format: worldname-cell-N)
	cellIndex := -1
	_, err := fmt.Sscanf(cellID, worldSpec.Name+"-cell-%d", &cellIndex)
	if err != nil || cellIndex < 0 {
		// Return empty bounds if we can't parse the cell ID
		return fleetforgev1.WorldBounds{}
	}

	// Recalculate boundaries for this specific cell
	cells := calculateCellBoundaries(worldSpec.Spec.Topology)
	if cellIndex < len(cells) {
		return cells[cellIndex]
	}

	// Return empty bounds if index is out of range
	return fleetforgev1.WorldBounds{}
}

// getPodHealthStatus determines health status based on pod conditions
func (r *WorldSpecReconciler) getPodHealthStatus(pod *corev1.Pod) string {
	// Check if pod is running and ready
	if pod.Status.Phase == corev1.PodRunning {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				if condition.Status == corev1.ConditionTrue {
					return "Healthy"
				} else {
					return "NotReady"
				}
			}
		}
		return "Starting"
	} else if pod.Status.Phase == corev1.PodPending {
		// Check if it's stuck in pending due to scheduling issues
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
				return "SchedulingFailed"
			}
		}
		return "Pending"
	} else if pod.Status.Phase == corev1.PodFailed {
		return "Failed"
	} else if pod.Status.Phase == corev1.PodSucceeded {
		return "Completed"
	}

	return "Unknown"
}
