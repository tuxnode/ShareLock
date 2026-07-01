package app_test

import (
	"testing"

	"github.com/cs161-staff/project2-starter-code/internal/client/app"
	userlib "github.com/cs161-staff/project2-starter-code/internal/userlib"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Client Tests")
}

const password = "password"
const aliceFile = "aliceFile.txt"
const bobFile = "bobFile.txt"
const charlesFile = "charlesFile.txt"
const contentOne = "Bitcoin is Nick's favorite "
const contentTwo = "digital "
const contentThree = "cryptocurrency!"

var _ = Describe("App Client", func() {

	var alice, bob, charles *app.Client
	var alicePhone, aliceLaptop, aliceDesktop *app.Client
	var err error

	BeforeEach(func() {
		userlib.DatastoreClear()
		userlib.KeystoreClear()
	})

	Describe("InitUser / GetUser", func() {

		It("should init and get a single user", func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())

			aliceLaptop = &app.Client{}
			err = aliceLaptop.GetUser("alice", password)
			Expect(err).To(BeNil())
		})

		It("should reject duplicate InitUser", func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())

			dup := &app.Client{}
			err = dup.InitUser("alice", password)
			Expect(err).ToNot(BeNil())
		})

		It("should reject GetUser with wrong password", func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())

			bob = &app.Client{}
			err = bob.GetUser("alice", "wrongpassword")
			Expect(err).ToNot(BeNil())
		})

		It("should reject GetUser for non-existent user", func() {
			alice = &app.Client{}
			err = alice.GetUser("nobody", password)
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("StoreFile / LoadFile", func() {

		BeforeEach(func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())
		})

		It("should store and load content", func() {
			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))
		})

		It("should store and load empty content", func() {
			err = alice.StoreFile(aliceFile, []byte{})
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte{}))
		})

		It("should reject LoadFile for non-existent file", func() {
			_, err = alice.LoadFile("nonexistent.txt")
			Expect(err).ToNot(BeNil())
		})

		It("should overwrite existing file on StoreFile", func() {
			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			err = alice.StoreFile(aliceFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentTwo)))
		})
	})

	Describe("AppendToFile", func() {

		BeforeEach(func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())
			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())
		})

		It("should append to an existing file", func() {
			err = alice.AppendToFile(aliceFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo)))
		})

		It("should support multiple appends", func() {
			err = alice.AppendToFile(aliceFile, []byte(contentTwo))
			Expect(err).To(BeNil())
			err = alice.AppendToFile(aliceFile, []byte(contentThree))
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo + contentThree)))
		})

		It("should reject AppendToFile for non-existent file", func() {
			err = alice.AppendToFile("nosuch.txt", []byte(contentTwo))
			Expect(err).ToNot(BeNil())
		})

		It("should reject AppendToFile with nil content", func() {
			err = alice.AppendToFile(aliceFile, nil)
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("Invitations (Create / Accept)", func() {

		BeforeEach(func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())
			bob = &app.Client{}
			err = bob.InitUser("bob", password)
			Expect(err).To(BeNil())
			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())
		})

		It("should share a file via invitation", func() {
			invite, err := alice.CreateInvitation(aliceFile, "bob")
			Expect(err).To(BeNil())

			err = bob.AcceptInvitation("alice", invite, bobFile)
			Expect(err).To(BeNil())

			data, err := bob.LoadFile(bobFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))
		})

		It("should allow shared user to append", func() {
			invite, err := alice.CreateInvitation(aliceFile, "bob")
			Expect(err).To(BeNil())

			err = bob.AcceptInvitation("alice", invite, bobFile)
			Expect(err).To(BeNil())

			err = bob.AppendToFile(bobFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo)))
		})

		It("should create invitation for non-existent file", func() {
			_, err = alice.CreateInvitation("nosuch.txt", "bob")
			Expect(err).ToNot(BeNil())
		})

		It("should reject accepting invitation for non-existent sender", func() {
			invite, err := alice.CreateInvitation(aliceFile, "bob")
			Expect(err).To(BeNil())

			err = bob.AcceptInvitation("eve", invite, bobFile)
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("Multi-session consistency", func() {

		BeforeEach(func() {
			aliceDesktop = &app.Client{}
			err = aliceDesktop.InitUser("alice", password)
			Expect(err).To(BeNil())
		})

		It("should see stored file from another session", func() {
			err = aliceDesktop.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			aliceLaptop = &app.Client{}
			err = aliceLaptop.GetUser("alice", password)
			Expect(err).To(BeNil())

			data, err := aliceLaptop.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))
		})

		It("should see appended content from another session", func() {
			err = aliceDesktop.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			alicePhone = &app.Client{}
			err = alicePhone.GetUser("alice", password)
			Expect(err).To(BeNil())

			err = alicePhone.AppendToFile(aliceFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			data, err := aliceDesktop.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo)))
		})

		It("should propagate invitations across sessions", func() {
			err = aliceDesktop.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			aliceLaptop = &app.Client{}
			err = aliceLaptop.GetUser("alice", password)
			Expect(err).To(BeNil())

			bob = &app.Client{}
			err = bob.InitUser("bob", password)
			Expect(err).To(BeNil())

			invite, err := aliceLaptop.CreateInvitation(aliceFile, "bob")
			Expect(err).To(BeNil())

			err = bob.AcceptInvitation("alice", invite, bobFile)
			Expect(err).To(BeNil())

			data, err := aliceDesktop.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))
		})
	})

	Describe("RevokeAccess", func() {

		BeforeEach(func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())
			bob = &app.Client{}
			err = bob.InitUser("bob", password)
			Expect(err).To(BeNil())
			charles = &app.Client{}
			err = charles.InitUser("charles", password)
			Expect(err).To(BeNil())

			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			invite, err := alice.CreateInvitation(aliceFile, "bob")
			Expect(err).To(BeNil())
			err = bob.AcceptInvitation("alice", invite, bobFile)
			Expect(err).To(BeNil())

			invite, err = bob.CreateInvitation(bobFile, "charles")
			Expect(err).To(BeNil())
			err = charles.AcceptInvitation("bob", invite, charlesFile)
			Expect(err).To(BeNil())
		})

		It("should revoke direct sharee", func() {
			err = alice.RevokeAccess(aliceFile, "bob")
			Expect(err).To(BeNil())

			_, err = bob.LoadFile(bobFile)
			Expect(err).ToNot(BeNil())
		})

		It("should keep owner access after revocation", func() {
			err = alice.RevokeAccess(aliceFile, "bob")
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))
		})

		It("should cascade revocation to indirect sharees", func() {
			err = alice.RevokeAccess(aliceFile, "bob")
			Expect(err).To(BeNil())

			_, err = charles.LoadFile(charlesFile)
			Expect(err).ToNot(BeNil())
		})

		It("should prevent revoked user from appending", func() {
			err = alice.RevokeAccess(aliceFile, "bob")
			Expect(err).To(BeNil())

			err = bob.AppendToFile(bobFile, []byte(contentTwo))
			Expect(err).ToNot(BeNil())
		})

		It("should allow owner to continue appending after revocation", func() {
			err = alice.RevokeAccess(aliceFile, "bob")
			Expect(err).To(BeNil())

			err = alice.AppendToFile(aliceFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			data, err := alice.LoadFile(aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo)))
		})
	})

	Describe("ListFiles", func() {

		BeforeEach(func() {
			alice = &app.Client{}
			err = alice.InitUser("alice", password)
			Expect(err).To(BeNil())
		})

		It("should list stored files", func() {
			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			files := alice.ListFiles()
			Expect(files).To(ContainElement(aliceFile))
		})

		It("should return empty list for new user", func() {
			files := alice.ListFiles()
			Expect(files).To(BeEmpty())
		})

		It("should list multiple files", func() {
			err = alice.StoreFile(aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())
			err = alice.StoreFile(bobFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			files := alice.ListFiles()
			Expect(files).To(HaveLen(2))
			Expect(files).To(ContainElement(aliceFile))
			Expect(files).To(ContainElement(bobFile))
		})
	})
})
