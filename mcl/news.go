package mcl

import (
	"context"
	"time"
)

// The addresses of the launcher content of Mojang.
const (
	// NewsURL serves the news about Minecraft.
	NewsURL = "https://launchercontent.mojang.com/news.json"
	// JavaPatchNotesURL serves the patch notes of Minecraft: Java Edition.
	JavaPatchNotesURL = "https://launchercontent.mojang.com/javaPatchNotes.json"
)

// News is the news about Minecraft.
//
// The format of the data may change at any time.
type News struct {
	// Version is the version of the format.
	Version int `json:"version"`
	// Entries are the news entries.
	Entries []NewsEntry `json:"entries"`
}

// NewsEntry is a single news entry.
type NewsEntry struct {
	// Title is the headline of the entry.
	Title string `json:"title"`
	// Category is the category of the entry.
	Category string `json:"category"`
	// Date is the day the entry was published.
	Date time.Time `json:"-"`
	// DateText is the date as it was published.
	DateText string `json:"date"`
	// Text is the body of the entry.
	Text string `json:"text"`
	// PlayPageImage is the image of the play page.
	PlayPageImage NewsImage `json:"playPageImage"`
	// NewsPageImage is the image of the news page.
	NewsPageImage NewsPageImage `json:"newsPageImage"`
	// ReadMoreLink is the address of the full entry.
	ReadMoreLink string `json:"readMoreLink"`
	// NewsType lists the types of the entry.
	NewsType []string `json:"newsType"`
	// ID is the identifier of the entry.
	ID string `json:"id"`
}

// NewsImage is an image of a news entry.
type NewsImage struct {
	// Title is the description of the image.
	Title string `json:"title"`
	// URL is the address of the image.
	URL string `json:"url"`
}

// NewsPageImage is the image of the news page.
type NewsPageImage struct {
	// Title is the description of the image.
	Title string `json:"title"`
	// URL is the address of the image.
	URL string `json:"url"`
	// Dimensions is the size of the image.
	Dimensions NewsImageDimensions `json:"dimensions"`
}

// NewsImageDimensions is the size of an image.
type NewsImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// JavaPatchNotes are the patch notes of Minecraft: Java Edition.
//
// The format of the data may change at any time.
type JavaPatchNotes struct {
	// Version is the version of the format.
	Version int `json:"version"`
	// Entries are the patch notes.
	Entries []JavaPatchNote `json:"entries"`
}

// JavaPatchNote is the patch notes of a single version.
type JavaPatchNote struct {
	// Title is the headline of the patch notes.
	Title string `json:"title"`
	// Type is "release" or "snapshot".
	Type string `json:"type"`
	// Version is the version the patch notes belong to.
	Version string `json:"version"`
	// Image is the image of the patch notes.
	Image NewsImage `json:"image"`
	// Body is the text of the patch notes.
	Body string `json:"body"`
	// ContentPath is the address of the full patch notes.
	ContentPath string `json:"contentPath"`
}

// GetMinecraftNews returns the news about Minecraft.
func GetMinecraftNews(ctx context.Context) (*News, error) {
	var news News
	if err := cachedJSON(ctx, NewsURL, &news); err != nil {
		return nil, err
	}
	for i := range news.Entries {
		if date, err := time.Parse("2006-01-02", news.Entries[i].DateText); err == nil {
			news.Entries[i].Date = date
		}
	}
	return &news, nil
}

// GetJavaPatchNotes returns the patch notes of Minecraft: Java Edition.
func GetJavaPatchNotes(ctx context.Context) (*JavaPatchNotes, error) {
	var notes JavaPatchNotes
	if err := cachedJSON(ctx, JavaPatchNotesURL, &notes); err != nil {
		return nil, err
	}
	return &notes, nil
}
