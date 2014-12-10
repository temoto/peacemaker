// +build linux
// +build cgo
package main

/*
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
*/
import "C"

var Sysconf_SC_CLK_TCK int

func init() {
	Sysconf_SC_CLK_TCK = int(C.sysconf(C._SC_CLK_TCK))
}
