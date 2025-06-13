from experience with go-swagger


* generate doc.go with package level documentation
* make sure single case switches revert to if
* make sure not unnecessary type conversion occurs
* make sure func signatures declare any unused parameter explicitly
* generate package level private helpers for error handling e.g. using errors.Is or errors.As


* IsNullable: may take the value "null" (valid)
* IsPointer: is a pointer and may be nil
* IsNillable: may take the value nil (pointers, slices and maps)

* mangling: do not replace "-" by "_" in package names

* enums should be a type, not just consts

* validations enrichment logic
* name deconfliction logic


1. Config loader
2. Schemas loader
