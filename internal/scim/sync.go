package scim

import (
	"fmt"
	"strings"

	"oauth-tester/internal/store"
)

// SyncResult tracks what happened during a push.
type SyncResult struct {
	UsersCreated  int      `json:"users_created"`
	UsersUpdated  int      `json:"users_updated"`
	UsersDeleted  int      `json:"users_deleted"`
	GroupsCreated int      `json:"groups_created"`
	GroupsDeleted int      `json:"groups_deleted"`
	MembersAdded  int      `json:"members_added"`
	MembersRemoved int    `json:"members_removed"`
	Errors        []string `json:"errors,omitempty"`
}

// Sync pushes local users and groups to Houston, creating/updating/deleting
// to make Houston match the local state.
func Sync(client *Client, s store.Store) (*SyncResult, error) {
	result := &SyncResult{}

	// 1. Sync users first (groups reference user IDs).
	userIDMap, err := syncUsers(client, s, result)
	if err != nil {
		return result, fmt.Errorf("sync users: %w", err)
	}

	// 2. Sync groups.
	if err := syncGroups(client, s, result, userIDMap); err != nil {
		return result, fmt.Errorf("sync groups: %w", err)
	}

	return result, nil
}

// syncUsers ensures Houston has the same users as local. Returns a map of
// local email -> Houston SCIM user ID for group membership sync.
func syncUsers(client *Client, s store.Store, result *SyncResult) (map[string]string, error) {
	localUsers, err := s.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("list local users: %w", err)
	}

	remoteUsers, err := client.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("list remote users: %w", err)
	}

	// Index remote users by userName (email), case-insensitive.
	remoteByEmail := make(map[string]SCIMUser)
	for _, u := range remoteUsers {
		remoteByEmail[strings.ToLower(u.UserName)] = u
	}

	// Map local email -> remote SCIM ID (populated as we create/match).
	userIDMap := make(map[string]string)

	// Track which remote users are accounted for.
	matched := make(map[string]bool)

	for _, local := range localUsers {
		email := strings.ToLower(local.Email)
		if email == "" {
			email = strings.ToLower(local.Username)
		}
		displayName := local.Name
		if displayName == "" {
			displayName = local.Username
		}

		if remote, ok := remoteByEmail[email]; ok {
			// User exists — update to make sure displayName matches.
			userIDMap[email] = remote.ID
			matched[email] = true

			if remote.DisplayName != displayName || !remote.Active {
				if err := client.UpdateUser(remote.ID, email, displayName); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("update user %s: %v", email, err))
				} else {
					result.UsersUpdated++
				}
			}
		} else {
			// User doesn't exist — create.
			created, err := client.CreateUser(email, displayName)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create user %s: %v", email, err))
			} else {
				userIDMap[email] = created.ID
				result.UsersCreated++
			}
		}
	}

	// Delete remote users that don't exist locally.
	for email, remote := range remoteByEmail {
		if !matched[email] && remote.Active {
			if err := client.DeleteUser(remote.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("delete user %s: %v", email, err))
			} else {
				result.UsersDeleted++
			}
		}
	}

	return userIDMap, nil
}

// syncGroups ensures Houston has the same groups and membership as local.
func syncGroups(client *Client, s store.Store, result *SyncResult, userIDMap map[string]string) error {
	localGroups, err := s.ListGroups()
	if err != nil {
		return fmt.Errorf("list local groups: %w", err)
	}

	remoteGroups, err := client.ListGroups()
	if err != nil {
		return fmt.Errorf("list remote groups: %w", err)
	}

	// Index remote groups by displayName.
	remoteByName := make(map[string]SCIMGroup)
	for _, g := range remoteGroups {
		remoteByName[g.DisplayName] = g
	}

	matched := make(map[string]bool)

	for _, local := range localGroups {
		members, err := s.ListGroupMembers(local.Name)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("list members of %s: %v", local.Name, err))
			continue
		}

		if remote, ok := remoteByName[local.Name]; ok {
			// Group exists — sync membership.
			matched[local.Name] = true
			syncGroupMembers(client, remote, members, userIDMap, s, result)
		} else {
			// Group doesn't exist — create, then add members.
			created, err := client.CreateGroup(local.Name)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create group %s: %v", local.Name, err))
				continue
			}
			result.GroupsCreated++

			// Add all members to the new group.
			var addIDs []string
			for _, username := range members {
				uid := resolveUserID(username, userIDMap, s)
				if uid != "" {
					addIDs = append(addIDs, uid)
				}
			}
			if len(addIDs) > 0 {
				if err := client.PatchGroupMembers(created.ID, addIDs, nil); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("add members to group %s: %v", local.Name, err))
				} else {
					result.MembersAdded += len(addIDs)
				}
			}
		}
	}

	// Delete remote groups that don't exist locally.
	for name, remote := range remoteByName {
		if !matched[name] {
			if err := client.DeleteGroup(remote.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("delete group %s: %v", name, err))
			} else {
				result.GroupsDeleted++
			}
		}
	}

	return nil
}

// syncGroupMembers diffs remote group members against local and patches.
func syncGroupMembers(client *Client, remote SCIMGroup, localMembers []string, userIDMap map[string]string, s store.Store, result *SyncResult) {
	// Build set of remote member IDs.
	remoteMemberIDs := make(map[string]bool)
	for _, m := range remote.Members {
		remoteMemberIDs[m.Value] = true
	}

	// Build set of desired member IDs.
	desiredIDs := make(map[string]bool)
	for _, username := range localMembers {
		uid := resolveUserID(username, userIDMap, s)
		if uid != "" {
			desiredIDs[uid] = true
		}
	}

	// Compute diff.
	var addIDs, removeIDs []string
	for uid := range desiredIDs {
		if !remoteMemberIDs[uid] {
			addIDs = append(addIDs, uid)
		}
	}
	for uid := range remoteMemberIDs {
		if !desiredIDs[uid] {
			removeIDs = append(removeIDs, uid)
		}
	}

	if len(addIDs) > 0 || len(removeIDs) > 0 {
		if err := client.PatchGroupMembers(remote.ID, addIDs, removeIDs); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("sync members of group %s: %v", remote.DisplayName, err))
		} else {
			result.MembersAdded += len(addIDs)
			result.MembersRemoved += len(removeIDs)
		}
	}
}

// resolveUserID finds the Houston SCIM user ID for a local username.
func resolveUserID(username string, userIDMap map[string]string, s store.Store) string {
	// Try email lookup first via the map.
	user, err := s.GetUser(username)
	if err != nil {
		return ""
	}
	email := strings.ToLower(user.Email)
	if email == "" {
		email = strings.ToLower(user.Username)
	}
	if id, ok := userIDMap[email]; ok {
		return id
	}
	return ""
}
