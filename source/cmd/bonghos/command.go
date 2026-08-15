package main

func commandAndArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "web", []string{"start"}
	}
	return args[0], args[1:]
}
