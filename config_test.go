package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	in := `host: 192.168.0.1
username: user
password: pass
`
	c, err := parse(strings.NewReader(in))
	assert.NoError(t, err)
	assert.EqualValues(t, &config{"192.168.0.1", "user", "pass"}, c)
}
