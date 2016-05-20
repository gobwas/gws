package client

func Go() error {
	c, err := getConfig()
	if err != nil {
		return err
	}

	if *scriptFile != "" {
		return GoLua(*scriptFile, c)
	} else {
		return GoCLI(c)
	}
}
