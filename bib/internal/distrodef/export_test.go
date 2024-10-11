package distrodef

import (
	"embed"
)

func MockDataDistroDefs(new embed.FS) (restore func()) {
	saved := dataDistroDefs
	dataDistroDefs = new
	return func() {
		dataDistroDefs = saved
	}
}
