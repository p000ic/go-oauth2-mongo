package mongo

import (
	"testing"
	"time"

	"github.com/go-oauth2/oauth2/v4/models"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTokenStore(t *testing.T) {
	Convey("Test mongodb token store", t, func() {
		store := NewTokenStore(NewConfig("mongodb://127.0.0.1:27017", "oauth2", username, password, "oauth2"))

		Convey("Test authorization code store", func() {
			info := &models.Token{
				ClientID:      "1",
				UserID:        "1_1",
				RedirectURI:   "http://localhost/",
				Scope:         "all",
				Code:          "11_11_11",
				CodeCreateAt:  time.Now(),
				CodeExpiresIn: time.Second * 5,
			}
			err := store.Create(store.ctx, info)
			So(err, ShouldBeNil)

			cinfo, err := store.GetByCode(info.Code)
			So(err, ShouldBeNil)
			So(cinfo.GetUserID(), ShouldEqual, info.UserID)

			err = store.RemoveByCode(info.Code)
			So(err, ShouldBeNil)

			cinfo, err = store.GetByCode(info.Code)
			So(err, ShouldBeNil)
			So(cinfo, ShouldBeNil)
		})

		Convey("Test access token store", func() {
			info := &models.Token{
				ClientID:        "1",
				UserID:          "1_1",
				RedirectURI:     "http://localhost/",
				Scope:           "all",
				Access:          "1_1_1",
				AccessCreateAt:  time.Now(),
				AccessExpiresIn: time.Second * 5,
			}
			err := store.Create(store.ctx, info)
			So(err, ShouldBeNil)

			ainfoT1, err := store.GetByAccess(info.GetAccess())
			So(err, ShouldBeNil)
			So(ainfoT1.GetUserID(), ShouldEqual, info.GetUserID())

			err = store.RemoveByAccess(info.GetAccess())
			So(err, ShouldBeNil)

			ainfoT2, err := store.GetByAccess(info.GetAccess())
			So(err, ShouldBeNil)
			So(ainfoT2, ShouldBeNil)
		})

		Convey("Test refresh token store", func() {
			info := &models.Token{
				ClientID:         "1",
				UserID:           "1_2",
				RedirectURI:      "http://localhost/",
				Scope:            "all",
				Access:           "1_2_1",
				AccessCreateAt:   time.Now(),
				AccessExpiresIn:  time.Second * 5,
				Refresh:          "1_2_2",
				RefreshCreateAt:  time.Now(),
				RefreshExpiresIn: time.Second * 15,
			}
			err := store.Create(store.ctx, info)
			So(err, ShouldBeNil)

			rinfoT1, err := store.GetByRefresh(info.GetRefresh())
			So(err, ShouldBeNil)
			So(rinfoT1.GetUserID(), ShouldEqual, info.GetUserID())

			err = store.RemoveByRefresh(info.GetRefresh())
			So(err, ShouldBeNil)

			rinfoT2, err := store.GetByRefresh(info.GetRefresh())
			So(err, ShouldBeNil)
			So(rinfoT2, ShouldBeNil)
		})

		Convey("Test clean-up store", func() {
			err := store.source.DropDatabase(store.ctx)
			So(err, ShouldBeNil)
		})
	})
}
