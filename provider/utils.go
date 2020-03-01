package provider

func getLogicParam(logic map[string]interface{}, key string) interface{} {
	if v, exists := logic[key]; exists {
		return v
	}

	return nil
}
