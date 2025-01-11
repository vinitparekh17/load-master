package main

import (
	"log"

	"github.com/go-playground/validator/v10"
)

func validateConfig() {
	validate := validator.New()
	validate.RegisterValidation("locationsMap", validateLocations)

	if err := validate.Struct(Config); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			log.Fatalf("Validation failed: Field '%s', Condition '%s'\n", err.Namespace(), err.Tag())
		}
		log.Fatal(err.Error())
	}
}

func validateLocations(fl validator.FieldLevel) bool {
	locations, ok := fl.Field().Interface().(map[string]location)
	if !ok {
		return false // Ensure the field is of the correct type
	}

	seenKeys := make(map[string]bool)

	for key := range locations {
		// Allow the single character "/" as valid
		if len(key) == 1 && key[0] == '/' {
			// Ensure keys are unique
			if seenKeys[key] {
				return false
			}
			seenKeys[key] = true
			continue
		}

		// For other keys, validate that they start and end with '/'
		if len(key) < 2 || key[0] != '/' || key[len(key)-1] != '/' {
			return false
		}

		// Ensure keys are unique
		if seenKeys[key] {
			return false
		}
		seenKeys[key] = true
	}
	return true
}
