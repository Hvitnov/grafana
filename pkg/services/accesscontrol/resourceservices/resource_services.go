package resourceservices

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/accesscontrol/resourcepermissions"
	"github.com/grafana/grafana/pkg/services/sqlstore"
)

func ProvideResourceServices(router routing.RouteRegister, sql *sqlstore.SQLStore, ac accesscontrol.AccessControl, store accesscontrol.ResourcePermissionsStore) (*ResourceServices, error) {
	teamPermissions, err := provideTeamPermissions(router, sql, ac, store)
	if err != nil {
		return nil, err
	}

	return &ResourceServices{services: map[string]*resourcepermissions.Service{
		"teams": teamPermissions,
	}}, nil
}

type ResourceServices struct {
	services map[string]*resourcepermissions.Service
}

func (s *ResourceServices) GetTeamService() *resourcepermissions.Service {
	return s.services["teams"]
}

var (
	TeamMemberActions = []string{
		"teams:read",
	}

	TeamAdminActions = []string{
		"teams:read",
		"teams:delete",
		"teams:write",
		"teams.permissions:read",
		"teams.permissions:write",
		"teams.preferences:read",
		"teams.preferences:write",
	}
)

func provideTeamPermissions(router routing.RouteRegister, sql *sqlstore.SQLStore, ac accesscontrol.AccessControl, store accesscontrol.ResourcePermissionsStore) (*resourcepermissions.Service, error) {
	options := resourcepermissions.Options{
		Resource:    "teams",
		OnlyManaged: true,
		ResourceValidator: func(ctx context.Context, orgID int64, resourceID string) error {
			id, err := strconv.ParseInt(resourceID, 10, 64)
			if err != nil {
				return err
			}

			err = sqlstore.GetTeamById(context.Background(), &models.GetTeamByIdQuery{
				OrgId: orgID,
				Id:    id,
			})
			if err != nil {
				return err
			}

			return nil
		},
		Assignments: resourcepermissions.Assignments{
			Users:        true,
			Teams:        false,
			BuiltInRoles: false,
		},
		PermissionsToActions: map[string][]string{
			"Member": TeamMemberActions,
			"Admin":  TeamAdminActions,
		},
		ReaderRoleName: "Team permission reader",
		WriterRoleName: "Team permission writer",
		RoleGroup:      "Teams",
		OnSetUser: func(ctx context.Context, orgID, userID int64, resourceID, permission string) error {
			switch permission {
			case "Member":
				// TODO: isExternal is used by team sync - check if team sync uses the endpoints for which these hooks have been added
				teamId, err := strconv.ParseInt(resourceID, 10, 64)
				if err != nil {
					return err
				}
				return sql.AddTeamMember(userID, orgID, teamId, false, 0)
			case "Admin":
				// TODO: isExternal is used by team sync - check if team sync uses the endpoints for which these hooks have been added
				teamId, err := strconv.ParseInt(resourceID, 10, 64)
				if err != nil {
					return err
				}
				return sql.AddTeamMember(userID, orgID, teamId, false, models.PERMISSION_ADMIN)
			default:
				return fmt.Errorf("invalid team permission type %d", permission)
			}
			return nil
		},
	}

	return resourcepermissions.New(options, router, ac, store)
}
