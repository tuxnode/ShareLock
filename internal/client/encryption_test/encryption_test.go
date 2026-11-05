package encryption_test

import (
	// Some imports use an underscore to prevent the compiler from complaining
	// about unused imports.
	_ "encoding/hex"
	_ "errors"
	_ "strconv"
	_ "strings"
	"testing"

	_ "github.com/google/uuid"

	// A "dot" import is used here so that the functions in the ginko and gomega
	// modules can be used without an identifier. For example, Describe() and
	// Expect() instead of ginko.Describe() and gomega.Describe().
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	client "github.com/tuxnode/ShareLock/internal/client/encryption"
)

func TestSetupAndExecution(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Tests")
}

func debugMsg(string, ...interface{}) {}

// ================================================
// Global Variables (feel free to add more!)
// ================================================
const defaultPassword = "password"
const emptyString = ""
const contentOne = "Bitcoin is Nick's favorite "
const contentTwo = "digital "
const contentThree = "cryptocurrency!"

// ================================================
// Describe(...) blocks help you organize your tests
// into functional categories. They can be nested into
// a tree-like structure.
// ================================================

// TestContext holds services and users for a test scenario
type TestContext struct {
	userService       *client.UserService
	fileService       *client.FileService
	invitationService *client.InvitationService

	alice        *client.User
	bob          *client.User
	charles      *client.User
	alicePhone   *client.User
	aliceLaptop  *client.User
	aliceDesktop *client.User
}

func newTestContext() *TestContext {
	storage := client.NewMemoryStorage()
	keyStore := client.NewMemoryKeyStore()
	return &TestContext{
		userService:       client.NewUserService(storage, keyStore),
		fileService:       client.NewFileService(storage, keyStore),
		invitationService: client.NewInvitationService(storage, keyStore),
	}
}

var _ = Describe("Client Tests", func() {

	var ctx *TestContext
	var err error

	aliceFile := "aliceFile.txt"
	bobFile := "bobFile.txt"
	charlesFile := "charlesFile.txt"

	BeforeEach(func() {
		ctx = newTestContext()
	})

	Describe("Basic Tests", func() {

		Specify("Basic Test: Testing InitUser/GetUser on a single user.", func() {
			debugMsg("Initializing user Alice.")
			ctx.alice, err = ctx.userService.InitUser("alice", defaultPassword)
			Expect(err).To(BeNil())

			debugMsg("Getting user Alice.")
			ctx.aliceLaptop, err = ctx.userService.GetUser("alice", defaultPassword)
			Expect(err).To(BeNil())
		})

		Specify("Basic Test: Testing Single User Store/Load/Append.", func() {
			debugMsg("Initializing user Alice.")
			ctx.alice, err = ctx.userService.InitUser("alice", defaultPassword)
			Expect(err).To(BeNil())

			debugMsg("Storing file data: %s", contentOne)
			err = ctx.fileService.StoreFile(ctx.alice, aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			debugMsg("Appending file data: %s", contentTwo)
			err = ctx.fileService.AppendToFile(ctx.alice, aliceFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			debugMsg("Appending file data: %s", contentThree)
			err = ctx.fileService.AppendToFile(ctx.alice, aliceFile, []byte(contentThree))
			Expect(err).To(BeNil())

			debugMsg("Loading file...")
			data, err := ctx.fileService.LoadFile(ctx.alice, aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo + contentThree)))
		})

		Specify("Basic Test: Testing Create/Accept Invite Functionality with multiple users and multiple instances.", func() {
			debugMsg("Initializing users Alice (aliceDesktop) and Bob.")
			ctx.aliceDesktop, err = ctx.userService.InitUser("alice", defaultPassword)
			Expect(err).To(BeNil())

			ctx.bob, err = ctx.userService.InitUser("bob", defaultPassword)
			Expect(err).To(BeNil())

			debugMsg("Getting second instance of Alice - aliceLaptop")
			ctx.aliceLaptop, err = ctx.userService.GetUser("alice", defaultPassword)
			Expect(err).To(BeNil())

			debugMsg("aliceDesktop storing file %s with content: %s", aliceFile, contentOne)
			err = ctx.fileService.StoreFile(ctx.aliceDesktop, aliceFile, []byte(contentOne))
			Expect(err).To(BeNil())

			debugMsg("aliceLaptop creating invite for Bob.")
			invite, err := ctx.invitationService.CreateInvitation(ctx.aliceLaptop, aliceFile, "bob")
			Expect(err).To(BeNil())

			debugMsg("Bob accepting invite from Alice under filename %s.", bobFile)
			err = ctx.invitationService.AcceptInvitation(ctx.bob, "alice", invite, bobFile)
			Expect(err).To(BeNil())

			debugMsg("Bob appending to file %s, content: %s", bobFile, contentTwo)
			err = ctx.fileService.AppendToFile(ctx.bob, bobFile, []byte(contentTwo))
			Expect(err).To(BeNil())

			debugMsg("aliceDesktop appending to file %s, content: %s", aliceFile, contentThree)
			err = ctx.fileService.AppendToFile(ctx.aliceDesktop, aliceFile, []byte(contentThree))
			Expect(err).To(BeNil())

			debugMsg("Checking that aliceDesktop sees expected file data.")
			data, err := ctx.fileService.LoadFile(ctx.aliceDesktop, aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo + contentThree)))

			debugMsg("Checking that aliceLaptop sees expected file data.")
			data, err = ctx.fileService.LoadFile(ctx.aliceLaptop, aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo + contentThree)))

			debugMsg("Checking that Bob sees expected file data.")
			data, err = ctx.fileService.LoadFile(ctx.bob, bobFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo + contentThree)))

			debugMsg("Getting third instance of Alice - alicePhone.")
			ctx.alicePhone, err = ctx.userService.GetUser("alice", defaultPassword)
			Expect(err).To(BeNil())

			debugMsg("Checking that alicePhone sees Alice's changes.")
			data, err = ctx.fileService.LoadFile(ctx.alicePhone, aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne + contentTwo + contentThree)))
		})

		Specify("Basic Test: Testing Revoke Functionality", func() {
			debugMsg("Initializing users Alice, Bob, and Charlie.")
			ctx.alice, err = ctx.userService.InitUser("alice", defaultPassword)
			Expect(err).To(BeNil())

			ctx.bob, err = ctx.userService.InitUser("bob", defaultPassword)
			Expect(err).To(BeNil())

			ctx.charles, err = ctx.userService.InitUser("charles", defaultPassword)
			Expect(err).To(BeNil())

			debugMsg("Alice storing file %s with content: %s", aliceFile, contentOne)
			ctx.fileService.StoreFile(ctx.alice, aliceFile, []byte(contentOne))

			debugMsg("Alice creating invite for Bob for file %s, and Bob accepting invite under name %s.", aliceFile, bobFile)

			invite, err := ctx.invitationService.CreateInvitation(ctx.alice, aliceFile, "bob")
			Expect(err).To(BeNil())

			err = ctx.invitationService.AcceptInvitation(ctx.bob, "alice", invite, bobFile)
			Expect(err).To(BeNil())

			debugMsg("Checking that Alice can still load the file.")
			data, err := ctx.fileService.LoadFile(ctx.alice, aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))

			debugMsg("Checking that Bob can load the file.")
			data, err = ctx.fileService.LoadFile(ctx.bob, bobFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))

			debugMsg("Bob creating invite for Charles for file %s, and Charlie accepting invite under name %s.", bobFile, charlesFile)
			invite, err = ctx.invitationService.CreateInvitation(ctx.bob, bobFile, "charles")
			Expect(err).To(BeNil())

			err = ctx.invitationService.AcceptInvitation(ctx.charles, "bob", invite, charlesFile)
			Expect(err).To(BeNil())

			debugMsg("Checking that Bob can load the file.")
			data, err = ctx.fileService.LoadFile(ctx.bob, bobFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))

			debugMsg("Checking that Charles can load the file.")
			data, err = ctx.fileService.LoadFile(ctx.charles, charlesFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))

			debugMsg("Alice revoking Bob's access from %s.", aliceFile)
			err = ctx.invitationService.RevokeAccess(ctx.alice, aliceFile, "bob")
			Expect(err).To(BeNil())

			debugMsg("Checking that Alice can still load the file.")
			data, err = ctx.fileService.LoadFile(ctx.alice, aliceFile)
			Expect(err).To(BeNil())
			Expect(data).To(Equal([]byte(contentOne)))

			debugMsg("Checking that Bob/Charles lost access to the file.")
			_, err = ctx.fileService.LoadFile(ctx.bob, bobFile)
			Expect(err).ToNot(BeNil())

			_, err = ctx.fileService.LoadFile(ctx.charles, charlesFile)
			Expect(err).ToNot(BeNil())

			debugMsg("Checking that the revoked users cannot append to the file.")
			err = ctx.fileService.AppendToFile(ctx.bob, bobFile, []byte(contentTwo))
			Expect(err).ToNot(BeNil())

			err = ctx.fileService.AppendToFile(ctx.charles, charlesFile, []byte(contentTwo))
			Expect(err).ToNot(BeNil())
		})

	})
})
