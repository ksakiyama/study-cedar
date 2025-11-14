package cedar

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/cedar-policy/cedar-go"
)

//go:embed policies/policy.cedar
var policyContent string

// Authorizer handles Cedar authorization
type Authorizer struct {
	policySet *cedar.PolicySet
}

// NewAuthorizer creates a new Cedar authorizer
func NewAuthorizer() (*Authorizer, error) {
	// Parse policies
	policySet, err := cedar.NewPolicySetFromBytes("policy.cedar", []byte(policyContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse policies: %w", err)
	}

	return &Authorizer{
		policySet: policySet,
	}, nil
}

// IsAuthorized checks if a user is authorized to perform an action on a resource
func (a *Authorizer) IsAuthorized(userID, userRole, action, resourceID, resourceOwnerID string) (bool, error) {
	// Create principal (user)
	principal := cedar.NewEntityUID(cedar.EntityType("DocumentApp::User"), cedar.String(userID))

	// Create action
	actionUID := cedar.NewEntityUID(cedar.EntityType("DocumentApp::Action"), cedar.String(action))

	// Create resource (document)
	resource := cedar.NewEntityUID(cedar.EntityType("DocumentApp::Document"), cedar.String(resourceID))

	// Build entities as JSON
	entitiesJSON := []map[string]interface{}{
		{
			"uid": map[string]string{
				"type": "DocumentApp::User",
				"id":   userID,
			},
			"attrs": map[string]interface{}{
				"role": userRole,
			},
			"parents": []interface{}{},
		},
	}

	// Add resource entity if it exists
	if resourceID != "" && resourceOwnerID != "" {
		entitiesJSON = append(entitiesJSON, map[string]interface{}{
			"uid": map[string]string{
				"type": "DocumentApp::Document",
				"id":   resourceID,
			},
			"attrs": map[string]interface{}{
				"owner": map[string]string{
					"type":       "DocumentApp::User",
					"id":         resourceOwnerID,
					"__entity":   "true",
				},
			},
			"parents": []interface{}{},
		})
	}

	// Convert to JSON and parse into EntityMap
	entitiesJSONBytes, err := json.Marshal(entitiesJSON)
	if err != nil {
		return false, fmt.Errorf("failed to marshal entities: %w", err)
	}

	var entities cedar.EntityMap
	if err := json.Unmarshal(entitiesJSONBytes, &entities); err != nil {
		return false, fmt.Errorf("failed to unmarshal entities: %w", err)
	}

	// Create request
	req := cedar.Request{
		Principal: principal,
		Action:    actionUID,
		Resource:  resource,
		Context:   cedar.NewRecord(cedar.RecordMap{}),
	}

	// Evaluate authorization
	decision, _ := a.policySet.IsAuthorized(entities, req)

	return decision == cedar.Allow, nil
}

// AuthzRequest represents an authorization request
type AuthzRequest struct {
	UserID          string
	UserRole        string
	Action          string
	ResourceID      string
	ResourceOwnerID string
}

// Authorize is a convenience method for authorization
func (a *Authorizer) Authorize(req AuthzRequest) (bool, error) {
	return a.IsAuthorized(req.UserID, req.UserRole, req.Action, req.ResourceID, req.ResourceOwnerID)
}
