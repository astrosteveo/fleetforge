import re

# Read the file
with open('pkg/cell/cell_test.go', 'r') as f:
    content = f.read()

# Pattern to find functions that need yMin/yMax variables
pattern = r'(func Test\w+\(t \*testing\.T\) \{)\n(\t[^y])'

def replacement(match):
    func_start = match.group(1)
    next_line = match.group(2)
    return f"{func_start}\n\tyMin := 0.0\n\tyMax := 1000.0\n{next_line}"

# Apply the pattern, but only to functions that don't already have yMin/yMax
functions_to_fix = [
    'TestCell_AddPlayer_AtCapacity',
    'TestCell_RemovePlayer', 
    'TestCell_UpdatePlayerPosition',
    'TestCell_GetPlayersInArea',
    'TestCell_Health',
    'TestCell_Metrics',
    'TestCell_CheckpointRestore'
]

for func_name in functions_to_fix:
    pattern = f'(func {func_name}\\(t \\*testing\\.T\\) \\{{)\n(\\tspec := CellSpec\\{{)'
    replacement_str = f'\\1\n\tyMin := 0.0\n\tyMax := 1000.0\n\\2'
    content = re.sub(pattern, replacement_str, content)

# Write back
with open('pkg/cell/cell_test.go', 'w') as f:
    f.write(content)

print("Fixed test functions")
