// Copyright © 2016 Asteris, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !solaris

package network_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/asteris-llc/converge/helpers/comparison"
	"github.com/asteris-llc/converge/helpers/fakerenderer"
	"github.com/asteris-llc/converge/helpers/logging"
	"github.com/asteris-llc/converge/resource"
	"github.com/asteris-llc/converge/resource/docker/network"
	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

// TestNetworkInterface verifies that Network implements the resource.Task
// interface
func TestNetworkInterface(t *testing.T) {
	t.Parallel()
	defer logging.HideLogs(t)()

	assert.Implements(t, (*resource.Task)(nil), new(network.Network))
}

// TestNetworkCheck tests the Network.Check function
func TestNetworkCheck(t *testing.T) {
	t.Parallel()
	defer logging.HideLogs(t)()

	nwName := "test-network"

	t.Run("state: absent", func(t *testing.T) {
		t.Run("network does not exist", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StateAbsent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(nil, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			require.NoError(t, err)
			assert.False(t, status.HasChanges())
			assert.Equal(t, 1, len(status.Diffs()))
			comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StateAbsent), string(network.StateAbsent))
		})

		t.Run("network exists", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StateAbsent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			require.NoError(t, err)
			assert.True(t, status.HasChanges())
			assert.Equal(t, 1, len(status.Diffs()))
			comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StateAbsent))
		})
	})

	t.Run("state: present", func(t *testing.T) {
		t.Run("network does not exist", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("ListNetworks").Return(nil, nil)
			c.On("FindNetwork", nwName).Return(nil, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			require.NoError(t, err)
			assert.True(t, status.HasChanges())
			assert.Equal(t, 1, len(status.Diffs()))
			comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StateAbsent), string(network.StatePresent))
		})

		t.Run("network exists", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("ListNetworks").Return(nil, nil)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			require.NoError(t, err)
			assert.False(t, status.HasChanges())
			assert.Equal(t, 1, len(status.Diffs()))
			comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
		})

		t.Run("gateway conflicts", func(t *testing.T) {
			nw := &network.Network{
				Name:  nwName,
				State: network.StatePresent,
				IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
					},
				},
			}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("ListNetworks").Return([]dc.Network{
				dc.Network{ID: "123", IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
					},
				}},
			}, nil)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			assert.Error(t, err)
			assert.Equal(t, fmt.Errorf("192.168.1.1: gateway(s) are already in use"), err)
			assert.Equal(t, resource.StatusCantChange, status.StatusCode())
		})

		t.Run("multiple different gateway conflicts", func(t *testing.T) {
			nw := &network.Network{
				Name:  nwName,
				State: network.StatePresent,
				IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
						dc.IPAMConfig{Gateway: "192.168.2.1"},
					},
				},
			}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("ListNetworks").Return([]dc.Network{
				dc.Network{ID: "123", IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
						dc.IPAMConfig{Gateway: "192.168.2.1"},
					},
				}},
			}, nil)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			assert.Error(t, err)
			assert.Equal(t, fmt.Errorf("192.168.1.1, 192.168.2.1: gateway(s) are already in use"), err)
			assert.Equal(t, resource.StatusCantChange, status.StatusCode())
		})

		t.Run("multiple same gateway conflicts", func(t *testing.T) {
			nw := &network.Network{
				Name:  nwName,
				State: network.StatePresent,
				IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
						dc.IPAMConfig{Gateway: "192.168.1.1"},
					},
				},
			}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("ListNetworks").Return([]dc.Network{
				dc.Network{ID: "123", IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
					},
				}},
			}, nil)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			assert.Error(t, err)
			assert.Equal(t, fmt.Errorf("192.168.1.1, 192.168.1.1: gateway(s) are already in use"), err)
			assert.Equal(t, resource.StatusCantChange, status.StatusCode())
		})

		t.Run("gateway multiple conflicts", func(t *testing.T) {
			nw := &network.Network{
				Name:  nwName,
				State: network.StatePresent,
				IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
					},
				},
			}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("ListNetworks").Return([]dc.Network{
				dc.Network{ID: "123", IPAM: dc.IPAMOptions{
					Config: []dc.IPAMConfig{
						dc.IPAMConfig{Gateway: "192.168.1.1"},
						dc.IPAMConfig{Gateway: "192.168.1.1"},
					},
				}},
			}, nil)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Check(context.Background(), fakerenderer.New())
			assert.Error(t, err)
			assert.Equal(t, fmt.Errorf("192.168.1.1, 192.168.1.1: gateway(s) are already in use"), err)
			assert.Equal(t, resource.StatusCantChange, status.StatusCode())
		})

		t.Run("network exists, force: true", func(t *testing.T) {
			t.Run("labels", func(t *testing.T) {
				nw := &network.Network{
					Name:   nwName,
					State:  network.StatePresent,
					Labels: map[string]string{"key": "val", "test": "val2"},
					Force:  true,
				}
				c := &mockClient{}
				nw.SetClient(c)
				c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)
				c.On("ListNetworks").Return(nil, nil)

				status, err := nw.Check(context.Background(), fakerenderer.New())
				require.NoError(t, err)
				assert.True(t, status.HasChanges())
				assert.True(t, len(status.Diffs()) > 1)
				comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
				comparison.AssertDiff(t, status.Diffs(), "labels", "", "key=val, test=val2")
			})

			t.Run("driver", func(t *testing.T) {
				nw := &network.Network{
					Name:   nwName,
					State:  network.StatePresent,
					Driver: "weave",
					Force:  true,
				}
				c := &mockClient{}
				nw.SetClient(c)
				c.On("FindNetwork", nwName).Return(&dc.Network{
					Name:   nwName,
					Driver: network.DefaultDriver,
				}, nil)
				c.On("ListNetworks").Return(nil, nil)

				status, err := nw.Check(context.Background(), fakerenderer.New())
				require.NoError(t, err)
				assert.True(t, status.HasChanges())
				assert.True(t, len(status.Diffs()) > 1)
				comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
				comparison.AssertDiff(t, status.Diffs(), "driver", network.DefaultDriver, "weave")
			})

			t.Run("options", func(t *testing.T) {
				nw := &network.Network{
					Name:    nwName,
					State:   network.StatePresent,
					Options: map[string]interface{}{"com.docker.network.bridge.enable_icc": "true"},
					Force:   true,
				}
				c := &mockClient{}
				nw.SetClient(c)
				c.On("FindNetwork", nwName).Return(&dc.Network{
					Name:    nwName,
					Options: nil,
				}, nil)
				c.On("ListNetworks").Return(nil, nil)

				status, err := nw.Check(context.Background(), fakerenderer.New())
				require.NoError(t, err)
				assert.True(t, status.HasChanges())
				assert.True(t, len(status.Diffs()) > 1)
				comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
				comparison.AssertDiff(t, status.Diffs(), "options", "", "com.docker.network.bridge.enable_icc=true")
			})

			t.Run("ipam options", func(t *testing.T) {
				nw := &network.Network{
					Name:  nwName,
					State: network.StatePresent,
					IPAM: dc.IPAMOptions{
						Driver: network.DefaultIPAMDriver,
						Config: []dc.IPAMConfig{
							dc.IPAMConfig{Subnet: "192.168.129.0/24"},
						},
					},
					Force: true,
				}
				c := &mockClient{}
				nw.SetClient(c)
				c.On("FindNetwork", nwName).Return(&dc.Network{
					Name: nwName,
				}, nil)
				c.On("ListNetworks").Return(nil, nil)

				status, err := nw.Check(context.Background(), fakerenderer.New())
				require.NoError(t, err)
				assert.True(t, status.HasChanges())
				assert.True(t, len(status.Diffs()) > 1)
				comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
				comparison.AssertDiff(t, status.Diffs(), "ipam_config", "", "subnet: 192.168.129.0/24")
			})

			t.Run("internal", func(t *testing.T) {
				nw := &network.Network{
					Name:     nwName,
					State:    network.StatePresent,
					Internal: true,
					Force:    true,
				}
				c := &mockClient{}
				nw.SetClient(c)
				c.On("FindNetwork", nwName).Return(&dc.Network{
					Name:     nwName,
					Internal: false,
				}, nil)
				c.On("ListNetworks").Return(nil, nil)

				status, err := nw.Check(context.Background(), fakerenderer.New())
				require.NoError(t, err)
				assert.True(t, status.HasChanges())
				assert.True(t, len(status.Diffs()) > 1)
				comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
				comparison.AssertDiff(t, status.Diffs(), "internal", "false", "true")
			})

			t.Run("ipv6", func(t *testing.T) {
				nw := &network.Network{
					Name:  nwName,
					State: network.StatePresent,
					IPv6:  true,
					Force: true,
				}
				c := &mockClient{}
				nw.SetClient(c)
				c.On("FindNetwork", nwName).Return(&dc.Network{
					Name:       nwName,
					EnableIPv6: false,
				}, nil)
				c.On("ListNetworks").Return(nil, nil)

				status, err := nw.Check(context.Background(), fakerenderer.New())
				require.NoError(t, err)
				assert.True(t, status.HasChanges())
				assert.True(t, len(status.Diffs()) > 1)
				comparison.AssertDiff(t, status.Diffs(), nwName, string(network.StatePresent), string(network.StatePresent))
				comparison.AssertDiff(t, status.Diffs(), "ipv6", "false", "true")
			})
		})
	})

	t.Run("docker api error", func(t *testing.T) {
		nw := &network.Network{Name: nwName, State: network.StatePresent}
		c := &mockClient{}
		nw.SetClient(c)
		c.On("FindNetwork", nwName).Return(nil, errors.New("error"))

		status, err := nw.Check(context.Background(), fakerenderer.New())
		require.Error(t, err)
		assert.Equal(t, resource.StatusFatal, status.StatusCode())
	})
}

// TestNetworkApply tests the Network.Apply function
func TestNetworkApply(t *testing.T) {
	t.Parallel()
	defer logging.HideLogs(t)()

	nwName := "test-network"

	t.Run("docker find network error", func(t *testing.T) {
		nw := &network.Network{Name: nwName}
		c := &mockClient{}
		nw.SetClient(c)
		c.On("FindNetwork", nwName).Return(nil, errors.New("error"))

		status, err := nw.Apply(context.Background())
		require.Error(t, err)
		assert.Equal(t, resource.StatusFatal, status.StatusCode())
	})

	t.Run("state: present", func(t *testing.T) {
		t.Run("network does not exist", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(nil, nil)
			c.On("CreateNetwork", mock.AnythingOfType("docker.CreateNetworkOptions")).
				Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Apply(context.Background())
			require.NoError(t, err)
			assert.True(t, status.HasChanges())
			c.AssertCalled(t, "CreateNetwork", mock.AnythingOfType("docker.CreateNetworkOptions"))
			c.AssertNotCalled(t, "RemoveNetwork", nwName)
		})

		t.Run("network exists, force: false", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Apply(context.Background())
			require.NoError(t, err)
			assert.False(t, status.HasChanges())
			c.AssertNotCalled(t, "CreateNetwork", mock.AnythingOfType("docker.CreateNetworkOptions"))
			c.AssertNotCalled(t, "RemoveNetwork", nwName)
		})

		t.Run("network exists, force: true", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent, Force: true}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)
			c.On("RemoveNetwork", nwName).Return(nil)
			c.On("CreateNetwork", mock.AnythingOfType("docker.CreateNetworkOptions")).
				Return(&dc.Network{Name: nwName}, nil)

			status, err := nw.Apply(context.Background())
			require.NoError(t, err)
			assert.True(t, status.HasChanges())
			c.AssertCalled(t, "RemoveNetwork", nwName)
			c.AssertCalled(t, "CreateNetwork", mock.AnythingOfType("docker.CreateNetworkOptions"))
		})

		t.Run("docker create network error", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(nil, nil)
			c.On("CreateNetwork", mock.AnythingOfType("docker.CreateNetworkOptions")).
				Return(nil, errors.New("error"))

			status, err := nw.Apply(context.Background())
			require.Error(t, err)
			assert.Equal(t, resource.StatusFatal, status.StatusCode())
		})

		t.Run("docker remove network error", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StatePresent, Force: true}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName, Driver: "test"}, nil)
			c.On("RemoveNetwork", mock.AnythingOfType("string")).Return(errors.New("error"))

			status, err := nw.Apply(context.Background())
			require.Error(t, err)
			assert.Equal(t, resource.StatusFatal, status.StatusCode())
		})
	})

	t.Run("state: absent", func(t *testing.T) {
		t.Run("network exists", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StateAbsent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)
			c.On("RemoveNetwork", nwName).Return(nil)

			status, err := nw.Apply(context.Background())
			require.NoError(t, err)
			assert.True(t, status.HasChanges())
			c.AssertCalled(t, "RemoveNetwork", nwName)
		})

		t.Run("network does not exist", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StateAbsent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(nil, nil)

			status, err := nw.Apply(context.Background())
			require.NoError(t, err)
			assert.False(t, status.HasChanges())
			c.AssertNotCalled(t, "RemoveNetwork", nwName)
		})

		t.Run("docker remove network error", func(t *testing.T) {
			nw := &network.Network{Name: nwName, State: network.StateAbsent}
			c := &mockClient{}
			nw.SetClient(c)
			c.On("FindNetwork", nwName).Return(&dc.Network{Name: nwName}, nil)
			c.On("RemoveNetwork", nwName).Return(errors.New("error"))

			status, err := nw.Apply(context.Background())
			require.Error(t, err)
			assert.Equal(t, resource.StatusFatal, status.StatusCode())
		})
	})
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) ListNetworks() ([]dc.Network, error) {
	args := m.Called()
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	return ret.([]dc.Network), args.Error(1)
}

func (m *mockClient) FindNetwork(name string) (*dc.Network, error) {
	args := m.Called(name)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	return ret.(*dc.Network), args.Error(1)
}

func (m *mockClient) CreateNetwork(opts dc.CreateNetworkOptions) (*dc.Network, error) {
	args := m.Called(opts)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	return ret.(*dc.Network), args.Error(1)
}

func (m *mockClient) RemoveNetwork(name string) error {
	args := m.Called(name)
	return args.Error(0)
}
