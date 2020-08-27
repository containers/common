package seccomp

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestGoArchToSeccompArchSuccess(t *testing.T) {
	for goArch, seccompArch := range goArchToSeccompArchMap {
		res, err := GoArchToSeccompArch(goArch)
		require.Nil(t, err)
		require.Equal(t, seccompArch, res)
	}
}

func TestGoArchToSeccompArchFailure(t *testing.T) {
	res, err := GoArchToSeccompArch("wrong")
	require.NotNil(t, err)
	require.Empty(t, res)
}

func TestSpecArchToSeccompArchSuccess(t *testing.T) {
	for specArch, seccompArch := range specArchToSeccompArchMap {
		res, err := specArchToSeccompArch(specArch)
		require.Nil(t, err)
		require.Equal(t, seccompArch, res)
	}
}

func TestSpecArchToSeccompArchFailure(t *testing.T) {
	res, err := specArchToSeccompArch("wrong")
	require.NotNil(t, err)
	require.Empty(t, res)
}

func TestSpecArchToLibseccompArchSuccess(t *testing.T) {
	for specArch, libseccompArch := range specArchToLibseccompArchMap {
		res, err := specArchToLibseccompArch(specArch)
		require.Nil(t, err)
		require.Equal(t, libseccompArch, res)
	}
}

func TestSpecArchToLibseccompArchFailure(t *testing.T) {
	res, err := specArchToLibseccompArch("wrong")
	require.NotNil(t, err)
	require.Empty(t, res)
}

func TestSpecActionToSeccompActionSuccess(t *testing.T) {
	for specAction, seccompAction := range specActionToSeccompActionMap {
		res, err := specActionToSeccompAction(specAction)
		require.Nil(t, err)
		require.Equal(t, seccompAction, res)
	}
}

func TestSpecActionToSeccompActionFailure(t *testing.T) {
	res, err := specActionToSeccompAction("wrong")
	require.NotNil(t, err)
	require.Empty(t, res)
}

func TestSpecOperatorToSeccompOperatorSuccess(t *testing.T) {
	for specOperator, seccompOperator := range specOperatorToSeccompOperatorMap {
		res, err := specOperatorToSeccompOperator(specOperator)
		require.Nil(t, err)
		require.Equal(t, seccompOperator, res)
	}
}

func TestSpecOperatorToSeccompOperatorFailure(t *testing.T) {
	res, err := specOperatorToSeccompOperator("wrong")
	require.NotNil(t, err)
	require.Empty(t, res)
}

func TestSpecToSeccomp(t *testing.T) {
	var ret uint = 1
	for _, tc := range []struct {
		input    *specs.LinuxSeccomp
		expected func(*Seccomp, error)
	}{

		{ // success
			input: &specs.LinuxSeccomp{
				DefaultAction: specs.ActKill,
				Architectures: []specs.Arch{
					specs.ArchX32,
					specs.ArchX86,
				},
				Syscalls: []specs.LinuxSyscall{
					{
						Names:    []string{"open", "rmdir"},
						Action:   specs.ActTrap,
						ErrnoRet: &ret,
						Args: []specs.LinuxSeccompArg{
							{
								Index:    0,
								Value:    20,
								ValueTwo: 10,
								Op:       specs.OpLessThan,
							},
							{
								Index:    1,
								Value:    10,
								ValueTwo: 12,
								Op:       specs.OpEqualTo,
							},
						},
					},
					{
						Names:    []string{"bind"},
						Action:   specs.ActTrap,
						ErrnoRet: &ret,
					},
				},
			},
			expected: func(profile *Seccomp, err error) {
				require.Nil(t, err)
				require.Equal(t, &Seccomp{
					DefaultAction: ActKill,
					Architectures: []Arch{ArchX32, ArchX86},
					Syscalls: []*Syscall{
						{
							Name:     "open",
							Action:   ActTrap,
							ErrnoRet: &ret,
							Args: []*Arg{
								{
									Index:    0,
									Value:    20,
									ValueTwo: 10,
									Op:       OpLessThan,
								},
								{
									Index:    1,
									Value:    10,
									ValueTwo: 12,
									Op:       OpEqualTo,
								},
							},
						},
						{
							Name:     "rmdir",
							Action:   ActTrap,
							ErrnoRet: &ret,
							Args: []*Arg{
								{
									Index:    0,
									Value:    20,
									ValueTwo: 10,
									Op:       OpLessThan,
								},
								{
									Index:    1,
									Value:    10,
									ValueTwo: 12,
									Op:       OpEqualTo,
								},
							},
						},
						{
							Name:     "bind",
							Action:   ActTrap,
							ErrnoRet: &ret,
							Args:     []*Arg{},
						},
					},
				}, profile)
			},
		},
		{ // wrong arch
			input: &specs.LinuxSeccomp{
				DefaultAction: specs.ActKill,
				Architectures: []specs.Arch{"wrong"},
			},
			expected: func(profile *Seccomp, err error) {
				require.NotNil(t, err)
				require.Nil(t, profile)
			},
		},
		{ // wrong op
			input: &specs.LinuxSeccomp{
				DefaultAction: specs.ActKill,
				Syscalls: []specs.LinuxSyscall{
					{
						Names:    []string{"rmdir"},
						Action:   specs.ActTrap,
						ErrnoRet: &ret,
						Args: []specs.LinuxSeccompArg{
							{Op: "wrong"},
						},
					},
				},
			},
			expected: func(profile *Seccomp, err error) {
				require.NotNil(t, err)
				require.Nil(t, profile)
			},
		},
		{ // wrong default action
			input: &specs.LinuxSeccomp{},
			expected: func(profile *Seccomp, err error) {
				require.NotNil(t, err)
				require.Nil(t, profile)
			},
		},
	} {
		tc.expected(specToSeccomp(tc.input))
	}
}
