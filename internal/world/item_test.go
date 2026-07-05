package world

import "testing"

func TestItemRegistry(t *testing.T) {
	torch, ok := GetItem("torch")
	if !ok {
		t.Fatal("expected torch in item registry")
	}
	if torch.Category != CategorySelectable {
		t.Errorf("torch category = %v, want CategorySelectable", torch.Category)
	}

	boots, ok := GetItem("pegasus_boots")
	if !ok {
		t.Fatal("expected pegasus_boots in item registry")
	}
	if boots.Category != CategoryPassive {
		t.Errorf("pegasus_boots category = %v, want CategoryPassive", boots.Category)
	}

	custom := ItemDef{ID: "custom_item", Name: "Custom Item", Category: CategoryPassive}
	RegisterItem(custom)

	got, ok := GetItem("custom_item")
	if !ok || got.Name != "Custom Item" {
		t.Errorf("failed to retrieve registered custom item, got %+v ok=%v", got, ok)
	}
}
