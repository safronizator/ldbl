# ldbl
Package ldbl (aka "loadable") is a simple DB's data access & ORM lib for Go, that's not using reflection or any other "magic". Also, it's not forcing some pattern (as such as Active Record or something else), but gives ability to developer to choose the most suitable approach to work with data in DB (you can start with loading data rows to simple structures, with data fields represented in dicts, and, later, add more complicated logic - like custom struct fields, methods, event callbacks, etc.).

## Disclaimer
Lib is under active development and not production-ready for the moment (some API may be added or changed). Use it on your own risk.
