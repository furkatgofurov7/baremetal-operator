package ironic

import (
	"net/http"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"
	"github.com/gophercloud/gophercloud/openstack/baremetalintrospection/v1/introspection"
	"github.com/stretchr/testify/assert"

	"github.com/metal3-io/baremetal-operator/pkg/bmc"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/clients"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/testserver"
)

func TestPowerOn(t *testing.T) {

	nodeUUID := "33ce8659-7400-4c68-9535-d10766f07a58"
	cases := []struct {
		name   string
		ironic *testserver.IronicMock

		expectedDirty        bool
		expectedError        bool
		expectedRequestAfter int
	}{
		{
			name: "node-already-power-on",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState: powerOn,
				UUID:       nodeUUID,
			}),
		},
		{
			name: "waiting-for-target-power-on",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:       powerOff,
				TargetPowerState: powerOn,
				UUID:             nodeUUID,
			}),
			expectedDirty:        true,
			expectedRequestAfter: 10,
		},
		{
			name: "power-on normal",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:           powerOff,
				TargetPowerState:     powerOff,
				TargetProvisionState: "",
				UUID:                 nodeUUID,
			}).WithNodeStatesPower(nodeUUID, http.StatusAccepted),
			expectedDirty: true,
		},
		{
			name: "power-on wait for Provisioning state",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:           powerOff,
				TargetPowerState:     powerOff,
				TargetProvisionState: string(nodes.TargetDeleted),
				UUID:                 nodeUUID,
			}),
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
		{
			name: "power-on wait for locked host",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:           powerOff,
				TargetPowerState:     powerOff,
				TargetProvisionState: "",
				UUID:                 nodeUUID,
			}).WithNodeStatesPower(nodeUUID, http.StatusConflict),
			expectedRequestAfter: 10,
			expectedDirty:        true,
			expectedError:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ironic != nil {
				tc.ironic.Start()
				defer tc.ironic.Stop()
			}

			inspector := testserver.NewInspector(t).Ready().WithIntrospection(nodeUUID, introspection.Introspection{
				Finished: false,
			})
			inspector.Start()
			defer inspector.Stop()

			host := makeHost()
			publisher := func(reason, message string) {}
			auth := clients.AuthConfig{Type: clients.NoAuth}
			prov, err := newProvisionerWithSettings(host, bmc.Credentials{}, publisher,
				tc.ironic.Endpoint(), auth, inspector.Endpoint(), auth,
			)
			if err != nil {
				t.Fatalf("could not create provisioner: %s", err)
			}

			prov.status.ID = nodeUUID
			result, err := prov.PowerOn()

			assert.Equal(t, tc.expectedDirty, result.Dirty)
			assert.Equal(t, time.Second*time.Duration(tc.expectedRequestAfter), result.RequeueAfter)
			if !tc.expectedError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestPowerOff(t *testing.T) {

	nodeUUID := "33ce8659-7400-4c68-9535-d10766f07a58"
	cases := []struct {
		name   string
		ironic *testserver.IronicMock

		expectedDirty        bool
		expectedError        bool
		expectedRequestAfter int
	}{
		{
			name: "node-already-power-off",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState: powerOff,
				UUID:       nodeUUID,
			}),
		},
		{
			name: "waiting-for-target-power-off",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:       powerOn,
				TargetPowerState: powerOff,
				UUID:             nodeUUID,
			}),
			expectedDirty:        true,
			expectedRequestAfter: 10,
		},
		{
			name: "power-off normal",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:           powerOn,
				TargetPowerState:     powerOn,
				TargetProvisionState: "",
				UUID:                 nodeUUID,
			}).WithNodeStatesPower(nodeUUID, http.StatusAccepted),
			expectedDirty: true,
		},
		{
			name: "power-off wait for Provisioning state",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:           powerOn,
				TargetPowerState:     powerOn,
				TargetProvisionState: string(nodes.TargetDeleted),
				UUID:                 nodeUUID,
			}),
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
		{
			name: "power-off wait for locked host",
			ironic: testserver.NewIronic(t).Ready().WithNode(nodes.Node{
				PowerState:           powerOn,
				TargetPowerState:     powerOn,
				TargetProvisionState: "",
				UUID:                 nodeUUID,
			}).WithNodeStatesPower(nodeUUID, http.StatusConflict),
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ironic != nil {
				tc.ironic.Start()
				defer tc.ironic.Stop()
			}

			inspector := testserver.NewInspector(t).Ready().WithIntrospection(nodeUUID, introspection.Introspection{
				Finished: false,
			})
			inspector.Start()
			defer inspector.Stop()

			host := makeHost()
			publisher := func(reason, message string) {}
			auth := clients.AuthConfig{Type: clients.NoAuth}
			prov, err := newProvisionerWithSettings(host, bmc.Credentials{}, publisher,
				tc.ironic.Endpoint(), auth, inspector.Endpoint(), auth,
			)
			if err != nil {
				t.Fatalf("could not create provisioner: %s", err)
			}

			prov.status.ID = nodeUUID
			result, err := prov.PowerOff()

			assert.Equal(t, tc.expectedDirty, result.Dirty)
			assert.Equal(t, time.Second*time.Duration(tc.expectedRequestAfter), result.RequeueAfter)
			if !tc.expectedError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
