package admin_test

import (
	"gateway/admin"
	"gateway/model"
	"gateway/model/testing"
	"gateway/stats"
	"regexp"

	gc "gopkg.in/check.v1"
)

func (a *AdminSuite) TestSampleBeforeValidate(c *gc.C) {
	acc1 := testing.PrepareAccount(c, a.db, testing.JeffAccount)
	user1 := testing.PrepareUser(c, a.db, acc1.ID, testing.JeffUser)
	api1 := testing.PrepareAPI(c, a.db, acc1.ID, user1.ID, testing.API1)

	acc2 := testing.PrepareAccount(c, a.db, testing.OtherAccount)
	user2 := testing.PrepareUser(c, a.db, acc2.ID, testing.OtherUser)
	api2 := testing.PrepareAPI(c, a.db, acc2.ID, user2.ID, testing.API2)

	controller := &admin.SamplesController{}
	for i, t := range []struct {
		should      string
		given       *model.Sample
		expectError string
	}{{
		should: "give error for invalid Operator",
		given: &model.Sample{
			Name:      "Sample1",
			AccountID: acc1.ID,
			UserID:    user1.ID,
			Constraints: []stats.Constraint{
				{Key: "api.id", Operator: "foo", Value: int64(5)},
			},
		},
		expectError: `invalid operator "foo" for single api.id value, use "EQ"`,
	}, {
		should: "give error for invalid value",
		given: &model.Sample{
			Name:      "Sample1",
			AccountID: acc1.ID,
			UserID:    user1.ID,
			Constraints: []stats.Constraint{
				{Key: "api.id", Operator: "EQ", Value: "5"},
			},
		},
		expectError: regexp.QuoteMeta("invalid type string for api.id value, must be int64 or []int64"),
	}, {
		should: "Work with a valid value",
		given: &model.Sample{
			Name:      "Sample1",
			AccountID: acc1.ID,
			UserID:    user1.ID,
			Constraints: []stats.Constraint{
				{Key: "api.id", Operator: stats.EQ, Value: api1.ID},
			},
		},
	}, {
		should: "disallow user1 from querying constraints api.id of api2",
		given: &model.Sample{
			Name:      "Sample3",
			AccountID: acc1.ID,
			UserID:    user1.ID,
			Constraints: []stats.Constraint{
				{Key: "api.id", Operator: stats.EQ, Value: api2.ID},
			},
		},
		expectError: regexp.QuoteMeta("api 2 not owned by user 1"),
	},
	} {
		c.Logf("test %d: should %s", i, t.should)

		tx, err := a.db.Begin()
		c.Assert(err, gc.IsNil)
		gotten := controller.BeforeValidate(t.given, tx)
		if t.expectError != "" {
			c.Check(gotten, gc.ErrorMatches, t.expectError)
		} else {
			c.Assert(gotten, gc.IsNil)
		}
		tx.Commit()

	}
}

func (a *AdminSuite) TestQueryStats(c *gc.C) {
	acc1 := testing.PrepareAccount(c, a.db, testing.JeffAccount)
	user1 := testing.PrepareUser(c, a.db, acc1.ID, testing.JeffUser)

	controller := &admin.SamplesController{}
	for i, t := range []struct {
		should      string
		given       *model.Sample
		expectError string
	}{{
		should: "give error for invalid Operator",
		given: &model.Sample{
			Name:      "Sample1",
			AccountID: acc1.ID,
			UserID:    user1.ID,
			Variables: []string{"request.size", "request.id"},
			Constraints: []stats.Constraint{
				{Key: "api.id", Operator: "foo", Value: int64(5)},
			},
		},
		expectError: `invalid operator "foo" for single api.id value, use "EQ"`,
	},
	} {
		c.Logf("test %d: should %s", i, t.should)

		tx, err := a.db.Begin()
		c.Assert(err, gc.IsNil)
		gotten, er := controller.QueryStats(t.given, tx)
		if t.expectError != "" {
			c.Check(er, gc.ErrorMatches, t.expectError)
		} else {
			c.Assert(gotten, gc.IsNil)
		}
		tx.Commit()

	}
}
