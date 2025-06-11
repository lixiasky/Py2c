import ast
import json
import sys

def ast_to_dict(node):
    if isinstance(node, ast.AST):
        result = {'_type': node.__class__.__name__}
        for field in node._fields:
            result[field] = ast_to_dict(getattr(node, field))
        for attr in node._attributes:
            result[attr] = ast_to_dict(getattr(node, attr, None))
        return result
    elif isinstance(node, list):
        return [ast_to_dict(x) for x in node]
    else:
        return node

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print(f'Usage: python {sys.argv[0]} <python_source_file>', file=sys.stderr)
        sys.exit(1)
    with open(sys.argv[1], 'r', encoding='utf-8') as f:
        source = f.read()
    tree = ast.parse(source, filename=sys.argv[1], mode='exec', type_comments=True)
    ast_dict = ast_to_dict(tree)
    json.dump(ast_dict, sys.stdout, indent=2, ensure_ascii=False) 