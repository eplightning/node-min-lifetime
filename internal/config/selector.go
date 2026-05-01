package config

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
)

type KubernetesSelector struct {
	Selector labels.Selector
}

func (s *KubernetesSelector) UnmarshalMapstructure(input any) error {
	str, ok := input.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", input)
	}

	selector, err := labels.Parse(str)
	if err != nil {
		return fmt.Errorf("invalid selector %w", err)
	}

	*s = KubernetesSelector{
		Selector: selector,
	}
	return nil
}
