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
)

// WorldSpecReconciler reconciles a WorldSpec object
type WorldSpecReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
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
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get WorldSpec")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling WorldSpec", "phase", worldSpec.Status.Phase)

	// Set initial phase if not set
	if worldSpec.Status.Phase == "" {
		worldSpec.Status.Phase = fleetforgev1.WorldSpecPhaseCreating
		if err := r.updateStatus(ctx, worldSpec, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle creation/update of cell pods
	result, err := r.reconcileCells(ctx, worldSpec, log)
	if err != nil {
		log.Error(err, "Failed to reconcile cells")
		worldSpec.Status.Phase = "Error"
		worldSpec.Status.Message = err.Error()
		if statusErr := r.updateStatus(ctx, worldSpec, log); statusErr != nil {
			log.Error(statusErr, "Failed to update status after error")
		}
		return result, err
	}

	// Update the status based on current state
	if err := r.updateWorldSpecStatus(ctx, worldSpec, log); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue for status updates
	return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
}

// reconcileCells manages the lifecycle of cell pods
func (r *WorldSpecReconciler) reconcileCells(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) (ctrl.Result, error) {
	// Calculate cell boundaries based on world topology
	cells := calculateCellBoundaries(worldSpec.Spec.Topology)

	for i, cellBounds := range cells {
		cellID := fmt.Sprintf("%s-cell-%d", worldSpec.Name, i)

		// Create or update deployment for this cell
		if err := r.reconcileCellDeployment(ctx, worldSpec, cellID, cellBounds, log); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile cell deployment %s: %w", cellID, err)
		}

		// Create or update service for this cell
		if err := r.reconcileCellService(ctx, worldSpec, cellID, log); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile cell service %s: %w", cellID, err)
		}
	}

	return ctrl.Result{}, nil
}

// reconcileCellDeployment creates or updates a deployment for a cell
func (r *WorldSpecReconciler) reconcileCellDeployment(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, cellID string, bounds fleetforgev1.WorldBounds, log logr.Logger) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cellID,
			Namespace: worldSpec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
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
		return err
	}

	log.Info("Successfully reconciled deployment", "deployment", cellID)
	return nil
}

// reconcileCellService creates or updates a service for a cell
func (r *WorldSpecReconciler) reconcileCellService(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, cellID string, log logr.Logger) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cellID + "-service",
			Namespace: worldSpec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
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
		return err
	}

	log.Info("Successfully reconciled service", "service", cellID+"-service")
	return nil
}

// updateWorldSpecStatus updates the status of the WorldSpec
func (r *WorldSpecReconciler) updateWorldSpecStatus(ctx context.Context, worldSpec *fleetforgev1.WorldSpec, log logr.Logger) error {
	// Count active cells by looking at deployments
	deploymentList := &appsv1.DeploymentList{}
	err := r.List(ctx, deploymentList, client.InNamespace(worldSpec.Namespace), client.MatchingLabels{"world": worldSpec.Name})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	activeCells := int32(0)
	expectedCells := worldSpec.Spec.Topology.InitialCells
	var cellStatuses []fleetforgev1.CellStatus

	for _, deployment := range deploymentList.Items {
		cellStatus := fleetforgev1.CellStatus{
			ID:         deployment.Name,
			PodName:    "", // Will be filled if pod exists
			Health:     "Unknown",
			Boundaries: fleetforgev1.WorldBounds{}, // Would need to be calculated from deployment
		}

		if deployment.Status.ReadyReplicas > 0 {
			activeCells++
			cellStatus.Health = "Healthy"
		} else {
			cellStatus.Health = "Pending"
		}

		cellStatuses = append(cellStatuses, cellStatus)
	}

	// Update basic status fields
	worldSpec.Status.ActiveCells = activeCells
	worldSpec.Status.Cells = cellStatuses
	worldSpec.Status.LastUpdateTime = &metav1.Time{Time: time.Now()}

	// Determine if world is ready (all expected cells are active)
	isReady := activeCells >= expectedCells && expectedCells > 0
	wasReady := r.isWorldReady(worldSpec)

	// Update phase based on readiness
	if isReady {
		worldSpec.Status.Phase = fleetforgev1.WorldSpecPhaseRunning
		worldSpec.Status.Message = fmt.Sprintf("World running with %d/%d active cells", activeCells, expectedCells)
	} else {
		worldSpec.Status.Phase = fleetforgev1.WorldSpecPhaseCreating
		worldSpec.Status.Message = fmt.Sprintf("Waiting for cells to become ready (%d/%d)", activeCells, expectedCells)
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
