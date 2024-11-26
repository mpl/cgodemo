module github.com/mpl/cgodemo/cgohello

go 1.22.2

require (
	github.com/mpl/cgodemo/libhello v0.1.0
)

replace (
	github.com/mpl/cgodemo/libhello => ../libhello
)
