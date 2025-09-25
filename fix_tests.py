import re

# Read the file
with open('pkg/cell/cell_test.go', 'r') as f:
    content = f.read()

# Find all functions that use YMin: &yMin but don't declare yMin
functions = re.findall(r'(func Test[^{]*\{[^}]*YMin: &yMin[^}]*)', content, re.DOTALL)

# For each function, add variable declarations if not present
def fix_function(match):
    func_content = match.group(0)
    if 'yMin := float64(0)' not in func_content:
        # Insert variable declarations after the opening brace
        lines = func_content.split('\n')
        for i, line in enumerate(lines):
            if line.strip().endswith('{'):
                lines.insert(i+1, '\tyMin := float64(0)')
                lines.insert(i+2, '\tyMax := float64(1000)')
                break
        return '\n'.join(lines)
    return func_content

# Apply fixes
fixed_content = re.sub(r'(func Test[^{]*\{[^}]*YMin: &yMin[^}]*)', fix_function, content, flags=re.DOTALL)

# Write back
with open('pkg/cell/cell_test.go', 'w') as f:
    f.write(fixed_content)
