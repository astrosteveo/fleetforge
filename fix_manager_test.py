import re

with open('pkg/cell/manager_test.go', 'r') as f:
    content = f.read()

# Pattern to find functions that use &yMin but don't declare it
pattern = r'(func Test[^{]*\{[^}]*?)(\s+)(.*?Boundaries: v1\.WorldBounds\{[^}]*YMin: &yMin)'

def fix_match(match):
    func_start = match.group(1)
    whitespace = match.group(2) 
    rest = match.group(3)
    
    # Check if variables are already declared
    if 'yMin := float64(' in func_start or 'yMin := float64(' in rest:
        return match.group(0)
    
    # Add variable declarations
    return func_start + whitespace + 'yMin := float64(0)\n\tyMax := float64(1000)\n\t' + rest

# Apply fix
fixed_content = re.sub(pattern, fix_match, content, flags=re.DOTALL)

with open('pkg/cell/manager_test.go', 'w') as f:
    f.write(fixed_content)
