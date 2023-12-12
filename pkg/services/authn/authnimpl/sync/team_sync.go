package sync

import (
	"context"
	"errors"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/authn"
	"github.com/grafana/grafana/pkg/services/dashboards/dashboardaccess"
	"github.com/grafana/grafana/pkg/services/team"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideTeamSync(teamService team.Service, userService user.Service) *TeamSync {
	return &TeamSync{teamService, userService, log.New("team.sync")}
}

type TeamSync struct {
	teamService team.Service
	userService user.Service

	log log.Logger
}

func (s *TeamSync) SyncTeamsHook(ctx context.Context, id *authn.Identity, _ *authn.Request) error {
	ctxLogger := s.log.FromContext(ctx)
	ctxLogger.Info("Synching User's Teams", "id", id.ID)

	if !id.ClientParams.SyncTeams {
		return nil
	}

	namespace, userID := id.NamespacedID()
	if namespace != authn.NamespaceUser || userID <= 0 {
		ctxLogger.Warn("Failed to sync teams, invalid namespace for identity", "id", id.ID, "namespace", namespace)
		return errors.New("invalid namespace for identity")
	}

	ctxLogger.Debug("Syncing teams", "id", id.ID, "Teams", id.Teams)
	// don't sync org roles if none is specified
	if len(id.Teams) == 0 {
		ctxLogger.Debug("Not syncing teams since external user doesn't have any", "id", id.ID)
		return nil
	}

	result, err := s.teamService.GetTeamIDsByUser(ctx, &team.GetTeamIDsByUserQuery{OrgID: id.OrgID, UserID: userID})
	if err != nil {
		ctxLogger.Error("Failed to get user's teams", "id", id.ID, "orgid", id.OrgID, "error", err)
		return nil
	}

	handledTeamIds := map[int64]bool{}
	deleteTeamIds := []int64{}

	// ignore any that are already existing
	for _, teamID := range result {
		handledTeamIds[teamID] = true
	}
	// add any new teams
	for _, teamID := range id.Teams {
		if _, ok := handledTeamIds[teamID]; !ok {
			//TODO how do we set isExternal?
			err := s.teamService.AddTeamMember(userID, id.OrgID, teamID, true, dashboardaccess.PERMISSION_VIEW)
			if err != nil {
				ctxLogger.Error("Failed to add user to team", "id", id.ID, "teamid", teamID, "error", err)
			}
			ctxLogger.Info("Added user to team", "id", id.ID, "teamid", teamID)
		}
	}

	// delete any removed teams
	for _, teamID := range result {
		found := false
		for _, newTeamId := range id.Teams {
			if teamID == newTeamId {
				found = true
				break
			}
		}
		if !found {
			err := s.teamService.RemoveTeamMember(ctx, &team.RemoveTeamMemberCommand{TeamID: teamID, UserID: userID, OrgID: id.OrgID})
			if err != nil {
				ctxLogger.Error("Failed to remove user from team", "id", id.ID, "teamid", teamID, "error", err)
			}
			ctxLogger.Info("Removed user from a team", "id", id.ID, "teamid", teamID)
			deleteTeamIds = append(deleteTeamIds, teamID)
		}
	}

	return nil
}
