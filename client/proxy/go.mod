module proxy

go 1.22

require (
	agent-scanner v0.0.0
	connect v0.0.0
	github.com/glebarez/go-sqlite v1.21.2
	github.com/robfig/cron/v3 v3.0.1
	knowledge v0.0.0
	runtimepaths v0.0.0
	skill-scanner v0.0.0
	static-server v0.0.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)

replace agent-scanner => ../agent

replace connect => ../connect

replace knowledge => ../knowledge

replace runtimepaths => ../runtimepaths

replace skill-scanner => ../skills

replace static-server => ../static
