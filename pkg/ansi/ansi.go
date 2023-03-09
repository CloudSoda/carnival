package ansi

import "runtime"

var reset = "\033[0m"
var red = "\033[31m"
var green = "\033[32m"
var yellow = "\033[33m"
var blue = "\033[34m"
var purple = "\033[35m"
var cyan = "\033[36m"
var gray = "\033[37m"
var white = "\033[97m"

func init() {
	if runtime.GOOS == "windows" {
		reset = ""
		red = ""
		green = ""
		yellow = ""
		blue = ""
		purple = ""
		cyan = ""
		gray = ""
		white = ""
	}
}

func Blue(in string) string {
	return blue + in + reset
}

func Cyan(in string) string {
	return cyan + in + reset
}

func Green(in string) string {
	return green + in + reset
}

func Purple(in string) string {
	return purple + in + reset
}
