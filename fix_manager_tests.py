import re

# Read the file
with open('pkg/cell/manager_test.go', 'r') as f:
    content = f.read()

# Replace all YMin: 0, YMax: 1000 with pointer references
content = re.sub(r'YMin: 0, YMax: 1000,', 'YMin: &yMin, YMax: &yMax,', content)

# Add variable declarations to each test function
functions = [
    'TestCellManager_CreateCell',
    'TestCellManager_CreateCell_Duplicate', 
    'TestCellManager_DeleteCell',
    'TestCellManager_AddRemovePlayer',
    'TestCellManager_UpdatePlayerPosition',
    'TestCellManager_GetHealth',
    'TestCellManager_GetMetrics',
    'TestCellManager_GetCellStats',
    'TestCellManager_Checkpoint',
    'TestCellManager_GetPlayerSession'
]

for func_name in functions:
    pattern = f'(func {func_name}\\(t \\*testing\\.T\\) \\{{)\n(\\t[^y])'
    replacement_str = f'\\1\n\tyMin := 0.0\n\tyMax := 1000.0\n\\2'
    content = re.sub(pattern, replacement_str, content)

# Special case for TestCellManager_ListCells which has multiple cell creations
pattern = r'(func TestCellManager_ListCells\(t \*testing\.T\) \{)\n(\t[^y])'
replacement_str = r'\1\n\tyMin := 0.0\n\tyMax := 1000.0\n\2'
content = re.sub(pattern, replacement_str, content)

# Write back
with open('pkg/cell/manager_test.go', 'w') as f:
    f.write(content)

print("Fixed manager test functions")
