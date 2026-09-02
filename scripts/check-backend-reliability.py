#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
backend = root / 'apps/backend'
errors = []


def balanced_calls(text, token):
    start = 0
    while True:
        index = text.find(token, start)
        if index < 0:
            return
        pos = index + len(token)
        depth = 1
        in_string = False
        escaped = False
        while pos < len(text) and depth:
            char = text[pos]
            if in_string:
                if escaped:
                    escaped = False
                elif char == '\\':
                    escaped = True
                elif char == '"':
                    in_string = False
            else:
                if char == '"':
                    in_string = True
                elif char == '(':
                    depth += 1
                elif char == ')':
                    depth -= 1
            pos += 1
        yield index, text[index:pos]
        start = pos


fmt_calls = 0
wrapped_calls = 0
for path in backend.rglob('*.go'):
    if path.name.endswith('_test.go'):
        continue
    text = path.read_text()
    for index, call in balanced_calls(text, 'fmt.Errorf('):
        fmt_calls += 1
        if '%w' in call:
            wrapped_calls += 1
            continue
        arguments = call[call.find('(') + 1:-1]
        if re.search(r'(^|[,\s])(err|[A-Za-z_][A-Za-z0-9_]*Err)\s*(,|$)', arguments):
            line = text.count('\n', 0, index) + 1
            errors.append(f'{path.relative_to(root)}:{line}: error argument formatted without %w')
        if '.Error()' in arguments:
            line = text.count('\n', 0, index) + 1
            errors.append(f'{path.relative_to(root)}:{line}: error stringified inside fmt.Errorf')

store_root = backend / 'internal/store'
for path in store_root.glob('*.go'):
    if path.name.endswith('_test.go'):
        continue
    text = path.read_text()
    if 'context.Background()' in text or 'context.TODO()' in text:
        errors.append(f'{path.relative_to(root)}: store code must not replace caller context')
    function_starts = list(re.finditer(r'^func\s+\([^\n]+', text, re.MULTILINE))
    for offset, match in enumerate(function_starts):
        end = function_starts[offset + 1].start() if offset + 1 < len(function_starts) else len(text)
        block = text[match.start():end]
        header = block.split('{', 1)[0]
        if '.pool.' in block and 'context.Context' not in header:
            line = text.count('\n', 0, match.start()) + 1
            errors.append(f'{path.relative_to(root)}:{line}: database store method lacks context.Context')
    for match in re.finditer(r'\.pool\.(?:Query|QueryRow|Exec|Begin)\s*\(\s*([^,\)]+)', text):
        if match.group(1).strip() != 'ctx':
            line = text.count('\n', 0, match.start()) + 1
            errors.append(f'{path.relative_to(root)}:{line}: pool operation does not use caller ctx')

redis_path = backend / 'internal/redisclient/redisclient.go'
redis_text = redis_path.read_text()
for token in [
    'ErrInvalidTarget = errors.New(',
    'ErrUnavailable   = errors.New(',
    'sanitizeError(err.Error(), target), ErrUnavailable',
]:
    if token not in redis_text:
        errors.append(f'{redis_path.relative_to(root)} missing {token}')

redis_test = (backend / 'internal/redisclient/redisclient_test.go').read_text()
for token in ['errors.Is(err, ErrInvalidTarget)', 'errors.Is(err, ErrUnavailable)']:
    if token not in redis_test:
        errors.append(f'redisclient tests missing {token}')

db_path = backend / 'internal/db/db.go'
db_text = db_path.read_text()
if db_text.count('context.WithTimeout(context.Background(), cleanupTimeout)') < 2:
    errors.append('apps/backend/internal/db/db.go must bound connection cleanup and migration unlock contexts')
if 'const cleanupTimeout = 2 * time.Second' not in db_text:
    errors.append('apps/backend/internal/db/db.go missing bounded cleanup timeout')

if errors:
    print('backend reliability contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)

print(f'backend reliability contract: OK ({fmt_calls} fmt.Errorf calls audited, {wrapped_calls} wrap causes)')
