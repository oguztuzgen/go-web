package main

import "testing"

func TestPersonRepository(t *testing.T) {
	t.Run("Person Repository", func(t *testing.T) {
		expected := Person{"Oguz", 12, "123"}
		pr := &PersonRepository{ds: []Person{{"Oguz", 12, "123"}}}

		t.Run("should not crash", func(t *testing.T) {
			pr.find(1)
		})

		t.Run("find all", func(t *testing.T) {
			people := pr.findAll()
			if len(people) == 0 {
				t.Errorf("expected people to have at least one value")
			}
		})

		t.Run("should find person oguz", func(t *testing.T) {
			got, err := pr.find(0)
			if err != nil || got != expected {
				t.Errorf("error: %q, expected %q, got %q\n", err, expected, got)
			}
		})

		t.Run("should save person ege", func(t *testing.T) {
			if err := pr.save(Person{"Ege", 15, "123"}); err != nil {
				t.Errorf("error: %q", err)
			}
		})

		t.Run("should find person ege", func(t *testing.T) {
			pr.save(Person{"Ege", 15, "123"})

			p2 := Person{"Ege", 15, "123"}
			if got, err := pr.find(1); got != p2 || err != nil {
				t.Errorf("error: %q, expected %q, got %q", err, p2, got)
			}

		})
	})
}

func TestPersonService(t *testing.T) {
	
}
