package security

// Authorizer checks if a user is allowed to interact with the bot.
type Authorizer struct {
	allowedIDs map[string]bool
}

// NewAuthorizer creates an authorizer with the given allowed user IDs.
// If the list is empty, all users are allowed.
func NewAuthorizer(allowedIDs []string) *Authorizer {
	m := make(map[string]bool, len(allowedIDs))
	for _, id := range allowedIDs {
		m[id] = true
	}
	return &Authorizer{allowedIDs: m}
}

// IsAllowed returns true if the user is authorized.
func (a *Authorizer) IsAllowed(userID string) bool {
	if len(a.allowedIDs) == 0 {
		return true // no allowlist = allow all
	}
	return a.allowedIDs[userID]
}
