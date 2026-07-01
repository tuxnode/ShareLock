package encryption

import (
	"testing"

	userlib "github.com/cs161-staff/project2-starter-code/internal/userlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSetupAndExecution(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Unit Tests")
}

var _ = Describe("Client Unit Tests", func() {

	BeforeEach(func() {
		userlib.DatastoreClear()
		userlib.KeystoreClear()
	})

	Describe("UserService", func() {
		Specify("InitUser sets the Username field correctly", func() {
			storage := NewMemoryStorage()
			keyStore := NewMemoryKeyStore()
			svc := NewUserService(storage, keyStore)

			alice, err := svc.InitUser("alice", "password")
			Expect(err).To(BeNil())
			Expect(alice.Username).To(Equal("alice"))
		})

		Specify("GetUser retrieves an existing user", func() {
			storage := NewMemoryStorage()
			keyStore := NewMemoryKeyStore()
			svc := NewUserService(storage, keyStore)

			_, err := svc.InitUser("alice", "password")
			Expect(err).To(BeNil())

			user, err := svc.GetUser("alice", "password")
			Expect(err).To(BeNil())
			Expect(user.Username).To(Equal("alice"))
		})

		Specify("GetUser rejects wrong password", func() {
			storage := NewMemoryStorage()
			keyStore := NewMemoryKeyStore()
			svc := NewUserService(storage, keyStore)

			_, err := svc.InitUser("alice", "password")
			Expect(err).To(BeNil())

			_, err = svc.GetUser("alice", "wrongpassword")
			Expect(err).ToNot(BeNil())
		})
	})
})
