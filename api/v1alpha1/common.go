package v1alpha1

func selectorForInstance(instance BookStack) map[string]string {
	return map[string]string{
		"app":                "bookstack",
		"bookstack-instance": instance.Name,
	}
}

// labelsForInstance is an alias for selectorForInstance
var labelsForInstance = selectorForInstance
