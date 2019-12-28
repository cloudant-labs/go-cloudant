/**
 * main - tests
 * - Test using `cloudanti` mock implementation
 */

package main

import (
	"testing"

	"github.com/barshociaj/go-cloudant/interface"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	// Set up
	client, _ := cloudanti.NewMockClient(cloudanti.CloudantContent{})

	// Run
	meta := create(client)

	//Evaluate
	assert.Equal(t, myItem.ID, meta.ID)
}
