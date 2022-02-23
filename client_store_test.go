package mongo

import (
	"testing"

	"github.com/go-oauth2/oauth2/v4/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestClientStore(t *testing.T) {
	store := NewClientStore(NewConfig("mongodb://127.0.0.1:27017", "oauth2", username, password, "oauth2"))

	client := &models.Client{
		ID:     "id",
		Secret: "secret",
		Domain: "domain",
		UserID: "user_id",
	}

	Convey("Set", t, func() {
		Convey("Success", func() {
			_ = store.RemoveByID(client.ID)
			err := store.Set(client)
			So(err, ShouldBeNil)
		})

		Convey("AlreadyExistingClient", func() {
			_ = store.RemoveByID(client.ID)
			_ = store.Set(client)
			err := store.Set(client)
			So(err, ShouldNotBeNil)
		})
	})

	Convey("GetByID", t, func() {
		Convey("Success", func() {
			_ = store.RemoveByID(client.ID)
			_ = store.Set(client)
			got, err := store.GetByID(client.ID)
			So(err, ShouldBeNil)
			So(got, ShouldResemble, client)
		})

		Convey("UnknownClient", func() {
			_, err := store.GetByID("unknown_client")
			So(err, ShouldNotBeNil)
		})
	})

	Convey("RemoveByID", t, func() {
		Convey("UnknownClient", func() {
			err := store.RemoveByID("unknown_client")
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Test clean-up store", t, func() {
		err := store.source.DropDatabase(store.ctx)
		So(err, ShouldBeNil)
	})
}
