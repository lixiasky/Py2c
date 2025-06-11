# Py2C — Python to C Transpiler (MVP)

Py2C is a lightweight MVP transpiler that converts Python 3.10 source code into valid C code using AST transformation. It is designed to demonstrate the feasibility of converting high-level language constructs to C while maintaining correctness and basic structure.

Note: Source code comments are primarily in Chinese, as this is an MVP stage prototype.

Features

## Supported

- Variables and expressions
  - Arithmetic: +, -, *, /, %, **
  - Comparison and logical operators
  - Automatic type inference: int, double, char

- Control flow
  - if / elif / else
  - while, for x in range(n)
  - break, continue, pass

- Functions
  - Definition and invocation
  - Return values
  - Type inference for parameters and return types

- print()
  - Supports multi-argument
  - Automatically chooses format specifier (%d, %f, %s)

- Classes and objects
  - class converted to struct
  - __init__, methods, attribute access
  - self mapped to struct pointer

- Lists
  - Simple list converted to C array (no slicing or append)

- Global code
  - All top-level code placed inside main()

## Not supported (output as comments in generated C code)

- try/except/finally
- with
- import, from ... import
- dict, set, tuple
- lambda, decorators, yield, async/await

## Usage

python3 py2ast.py example.py > example_ast.json
go run ast2c.go example_ast.json > example.c
gcc -o example example.c
./example

## Example

The included example.py demonstrates support for:
- Functions
- Class definition and usage
- Control structures
- Print output

The corresponding C output compiles via GCC and executes partially with correct output.

## Notes

- Some generated C files may require manually adding missing #include lines.
- Output structure is formatted with readable indentation and fallback comments.
- Only one test file (example.py) has been verified to compile and partially execute correctly.

## Contact

I'd be glad to receive feedback if you'd like.

Email:lixiasky+public@protonmail.com

## License

MIT License. See LICENSE for details.