package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListSetupExperienceStatusResults", testSetupExperienceStatusResults},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testSetupExperienceStatusResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hostUUID := uuid.NewString()

	// VPP setup: create a token so that we can insert a VPP app
	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Donkey Kong", "Jungle")
	require.NoError(t, err)
	tok1, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
	assert.NoError(t, err)
	vppApp, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{BundleIdentifier: "com.test.test", Name: "test.app", LatestVersion: "1.0.0"}, nil)
	require.NoError(t, err)
	var vppAppsTeamsID uint
	err = sqlx.GetContext(context.Background(), ds.reader(ctx),
		&vppAppsTeamsID, `SELECT id FROM vpp_apps_teams WHERE adam_id = ?`,
		vppApp.AdamID,
	)
	require.NoError(t, err)

	// Create a software installer
	// We need a new user first
	user, err := ds.NewUser(ctx, &fleet.User{Name: "Foo", Email: "foo@example.com", GlobalRole: ptr.String("admin"), Password: []byte("12characterslong!")})
	require.NoError(t, err)
	installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{Filename: "test.app", Version: "1.0.0", UserID: user.ID})
	require.NoError(t, err)

	// TODO: use DS methods once those are written
	var scriptID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO setup_experience_scripts (name) VALUES (?)`,
			"test_script")
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		scriptID = uint(id)
		return nil
	})

	// Software installer step
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, name, status, software_installer_id) VALUES (?, ?, ?, ?)`,
			hostUUID, "software", fleet.StatusPending, installerID)
		require.NoError(t, err)
		return nil
	})

	// VPP app step
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, name, status, vpp_app_team_id) VALUES (?, ?, ?, ?)`,
			hostUUID, "vpp", fleet.StatusPending, vppAppsTeamsID)
		require.NoError(t, err)
		return nil
	})

	// Setup script step
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, name, status, setup_experience_script_id) VALUES (?, ?, ?, ?)`,
			hostUUID, "script", fleet.StatusPending, scriptID)
		require.NoError(t, err)
		return nil
	})

	res, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID)
	require.NoError(t, err)
	require.Len(t, res, 3)
	for i, s := range []*fleet.SetupExperienceStatusResult{
		{
			ID:                  1,
			HostUUID:            hostUUID,
			Name:                "software",
			Status:              fleet.StatusPending,
			SoftwareInstallerID: ptr.Uint(installerID),
		},
		{
			ID:           2,
			HostUUID:     hostUUID,
			Name:         "vpp",
			Status:       fleet.StatusPending,
			VPPAppTeamID: ptr.Uint(vppAppsTeamsID),
		},
		{
			ID:                      3,
			HostUUID:                hostUUID,
			Name:                    "script",
			Status:                  fleet.StatusPending,
			SetupExperienceScriptID: ptr.Uint(scriptID),
		},
	} {
		require.Equal(t, s, res[i])
	}
}
