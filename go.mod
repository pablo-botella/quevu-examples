module github.com/pablo-botella/quevu-demo

go 1.26.0

require (
	github.com/pablo-botella/quevu-hub v0.0.0
	github.com/pablo-botella/quevu-kernel v0.0.0
	github.com/pablo-botella/quevu-txt v0.0.0
	github.com/pablo-botella/quevu-ui-go v0.0.0
	github.com/pablo-botella/quevu-wire v0.0.0
)

require github.com/pablo-botella/dbh v0.0.5 // indirect

require (
	github.com/Microsoft/go-winio v0.6.4 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/pablo-botella/linereader v0.5.2 // indirect
	github.com/pablo-botella/quevu-panel v0.0.0
	github.com/pablo-botella/quevu-sql v0.0.0
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.49.0 // indirect
	modernc.org/libc v1.80.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.62.0 // indirect
)

replace (
	github.com/pablo-botella/quevu-hub => ../quevu-hub
	github.com/pablo-botella/quevu-kernel => ../quevu-kernel
	github.com/pablo-botella/quevu-txt => ../quevu-txt
	github.com/pablo-botella/quevu-ui-go => ../quevu-ui-go
	github.com/pablo-botella/quevu-wire => ../quevu-wire
)

replace github.com/pablo-botella/quevu-panel => ../quevu-panel

replace github.com/pablo-botella/quevu-sql => ../quevu-sql
