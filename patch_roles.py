import sys
import re

file_path = 'ui-react/src/pages/RolesPage.tsx'
with open(file_path, 'r') as f:
    content = f.read()

# Regex for the input
# We need to be careful with multiline matching if the tag spans multiple lines
# The input tag in the file seems to span multiple lines
input_pattern = r'(<input\s+type="text"\s+list="new-role-resource-list"[\s\S]+?className=")([^"]+)("[\s\S]+?placeholder=")([^"]+)("\s*/>)'
input_replacement = r'\1flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring\3Select or type resource...\5'

# Regex for the select
select_pattern = r'(<select\s+value={newRolePolicy\.action}[\s\S]+?className=")([^"]+)("\s*>)'
select_replacement = r'\1flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring\3'

new_content = re.sub(input_pattern, input_replacement, content)
new_content = re.sub(select_pattern, select_replacement, new_content)

if new_content != content:
    with open(file_path, 'w') as f:
        f.write(new_content)
    print('Successfully patched file')
else:
    print('Could not find target code')
