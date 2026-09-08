package world

// lectern.go: canonical book insert/remove for minecraft:lectern.
//
// A lectern stores one book (written_book or writable_book) in slot 0 of its
// block entity Items list. The has_book block state property tracks whether a
// book is present. The powered property is reserved for the short page-turn
// pulse; comparators read the page directly from the block entity.

// IsLecternBook reports whether an item ID is placeable on a lectern.
func IsLecternBook(itemID string) bool {
	return itemID == "minecraft:written_book" || itemID == "minecraft:writable_book"
}

// LecternBook returns the item ID stored in the lectern's block entity, or "".
func LecternBook(entity BlockEntity) string {
	for _, item := range entity.Items {
		if item.Slot == 0 && item.ItemID != "" {
			return item.ItemID
		}
	}
	return ""
}

// LecternPageCount returns the number of pages in the readable book stored in
// slot zero. Books without preserved page components retain one readable page.
func LecternPageCount(items []ContainerItem) int {
	for _, item := range items {
		if item.Slot != 0 || !IsLecternBook(item.ItemID) {
			continue
		}
		stack := item.Stack()
		for _, component := range []string{
			"minecraft:written_book_content", "written_book_content",
			"minecraft:writable_book_content", "writable_book_content",
		} {
			var content struct {
				Pages []any `json:"pages"`
			}
			if stack.Component(component, &content) {
				return max(1, len(content.Pages))
			}
		}
		return 1
	}
	return 0
}

// InsertLecternBook places a book on an empty lectern. Returns the updated
// block (has_book=true, powered=false) and true on success.
func InsertLecternBook(block Block, itemID string) (Block, bool) {
	if block.ResourceLocation() != "minecraft:lectern" {
		return block, false
	}
	if block.Properties["has_book"] == "true" {
		return block, false
	}
	if !IsLecternBook(itemID) {
		return block, false
	}
	updated := copyWorldBlock(block)
	updated.Properties["has_book"] = "true"
	updated.Properties["powered"] = "false"
	return updated, true
}

// EjectLecternBook removes the book from a lectern. Returns the book item ID,
// the cleared block (has_book=false, powered=false), and true on success.
func EjectLecternBook(block Block, storedBook string) (itemID string, updated Block, ok bool) {
	if block.ResourceLocation() != "minecraft:lectern" {
		return "", block, false
	}
	if block.Properties["has_book"] != "true" || storedBook == "" {
		return "", block, false
	}
	cleared := copyWorldBlock(block)
	cleared.Properties["has_book"] = "false"
	cleared.Properties["powered"] = "false"
	return storedBook, cleared, true
}
