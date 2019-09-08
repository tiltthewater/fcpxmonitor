package main

import (
	"net/http"
	"testing"
	"time"
)

func Test_Latest(t *testing.T) {
	t1 := time.Now().Unix()
	time.Sleep(1 * time.Second)
	t2 := time.Now().Unix()
	latest := Latest(t1, t2)
	if latest != t2 {
		t.Fatalf("Did not get latest: %d <> %d", t1, latest)
	}
}

func Test_Library_HasKey(t *testing.T) {
	library := NewLibrary()
	if library.HasProject("1234") {
		t.Error("Library should not have key")
	}
}

func Test_Library_Checkout_Project(t *testing.T) {
	library := NewLibrary()
	err := library.CheckoutProject("1234", "Test Project", "host1", testFCPXBundlePath, map[string]string{})
	if err != nil {
		t.Error(err.Error())
	}
}

func Test_Library_Checkout_Project_Conflict(t *testing.T) {
	library := NewLibrary()
	err := library.CheckoutProject("1234", "Test Project", "host1", testFCPXBundlePath, map[string]string{})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = library.CheckoutProject("1234", "Test Project", "host2", testFCPXBundlePath, map[string]string{})
	if err == nil {
		t.Fatal("Should get an error because there's more than one host")
		checkouts := len(library.Projects["1234"].Checkouts)
		if checkouts != 2 {
			t.Fatalf("Expected 2 checkouts: %d\n", checkouts)
		}
	} else {
		t.Log(err.Error())
	}
}

func Test_Library_Deregister_Project(t *testing.T) {
	library := NewLibrary()
	err := library.CheckoutProject("1234", "Test Project", "host1", testFCPXBundlePath, map[string]string{})
	if err != nil {
		t.Error(err.Error())
	}
	uuids := map[string]bool{} // No open projects
	_, removed := library.DeregisterProjects("host1", uuids)
	if library.HasProject("1234") {
		t.Errorf("%v\n", library)
	}
	if len(removed) != 1 {
		t.Fatal("Expected 1 removed uuid")
	}
	if removed[0] != "1234" {
		t.Fatal("Expected '1234' in removed uuids")
	}
}

func Test_Library_Update_404(t *testing.T) {
	library := NewLibrary()
	status, _ := library.Update(ProjectUpdate{
		Hostname: "nobody",
		UUID:     "1234",
		Last:     time.Now().Unix(),
	})
	if status != http.StatusNotFound {
		t.Fatal("Expected 404")
	}
}

func Test_Library_Update_No_Checkout(t *testing.T) {
	library := NewLibrary()
	err := library.CheckoutProject("1234", "Test Project", "host1", testFCPXBundlePath, map[string]string{})
	if err != nil {
		t.Error(err.Error())
	}
	status, _ := library.Update(ProjectUpdate{
		Hostname: "host2",
		UUID:     "1234",
		Last:     time.Now().Unix(),
	})
	if status != http.StatusForbidden {
		t.Fatal("Expected 403")
	}
}

func Test_Library_Update(t *testing.T) {
	library := NewLibrary()
	err := library.CheckoutProject("1234", "Test Project", "host1", testFCPXBundlePath, map[string]string{})
	if err != nil {
		t.Error(err.Error())
	}
	time.Sleep(2 * time.Second)
	t1 := time.Now().Unix()
	status, err := library.Update(ProjectUpdate{
		Hostname: "host1",
		UUID:     "1234",
		Last:     t1,
	})
	if status != 0 {
		t.Fatal("Update failed: " + err.Error())
	}
	last := library.Projects["1234"].Checkouts["host1"].Last
	if last != t1 {
		t.Fatalf("Update failed check: %d | %d", t1, last)
	}
}
