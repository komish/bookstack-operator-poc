package controllers

import (
	toolsv1alpha1 "github.com/opdev/bookstack-operator/api/v1alpha1"
)

func selectorForInstance(instance toolsv1alpha1.BookStack) map[string]string {
	return map[string]string{
		"app":                "bookstack",
		"bookstack-instance": instance.Name,
	}
}

// labelsForInstance is an alias for selectorForInstance
var labelsForInstance = selectorForInstance
