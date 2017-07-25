package main

import (
	"log"
	"sort"
	"strings"
	"time"

	"github.com/xoebus/go-tracker"
)

const IssueLabelUnscheduled = "unscheduled"
const IssueLabelScheduled = "scheduled"
const IssueLabelInFlight = "in-flight"
const IssueLabelBug = "bug"
const IssueLabelEnhancement = "enhancement"

var storyStateLabels = map[string]string{
	IssueLabelUnscheduled: "e4eff7",
	IssueLabelScheduled:   "f4f4f4",
	IssueLabelInFlight:    "f3f3d1",

	// respect original github colors
	IssueLabelBug:         "",
	IssueLabelEnhancement: "",
}

var issueOnlyLabels = map[string]string{
	"discuss": "c2e0c6",
}

type StorySet []tracker.Story

func (set StorySet) WithLabel(label string) StorySet {
	var withLabel StorySet
	for _, story := range set {
		for _, storyLabel := range story.Labels {
			if strings.EqualFold(storyLabel.Name, label) {
				withLabel = append(withLabel, story)
				break
			}
		}
	}

	return withLabel
}

type dupeEquivalence struct {
	name        string
	description string
	labels      string
}

func (set StorySet) Dedupe() (StorySet, StorySet) {
	byEquivalence := map[dupeEquivalence]StorySet{}

	for _, story := range set {
		labelNames := []string{}
		for _, label := range story.Labels {
			labelNames = append(labelNames, label.Name)
		}

		sort.Strings(labelNames)

		eq := dupeEquivalence{
			name:        story.Name,
			description: story.Description,
			labels:      strings.Join(labelNames, ","),
		}

		byEquivalence[eq] = append(byEquivalence[eq], story)
	}

	deduped := StorySet{}
	dupes := StorySet{}

	for _, stories := range byEquivalence {
		if len(stories) == 1 {
			deduped = append(deduped, stories[0])
			continue
		}

		var oldestStory tracker.Story
		for _, story := range stories {
			if oldestStory.ID == 0 || story.ID < oldestStory.ID {
				oldestStory = story
			}
		}

		deduped = append(deduped, oldestStory)
		for _, story := range stories {
			if story.ID != oldestStory.ID {
				dupes = append(dupes, story)
			}
		}
	}

	return deduped, dupes
}

func (set StorySet) AllAccepted() bool {
	allAccepted := true
	for _, story := range set {
		if story.State != "accepted" {
			allAccepted = false
			break
		}
	}

	return allAccepted
}

func (set StorySet) Unscheduled() bool {
	for _, story := range set {
		if story.Type != tracker.StoryStateUnscheduled {
			return false
		}
	}

	return true
}

func (set StorySet) Untriaged() bool {
	for _, story := range set {
		if story.Type != tracker.StoryTypeChore {
			return false
		}
	}

	return true
}

func (set StorySet) HasPR() bool {
	for _, story := range set {
		for _, label := range story.Labels {
			if label.Name == "has-pr" {
				return true
			}
		}
	}

	return false
}

func (set StorySet) LastAccepted() time.Time {
	lastAccepted := time.Unix(0, 0)

	for _, story := range set {
		if story.AcceptedAt.After(lastAccepted) {
			lastAccepted = *story.AcceptedAt
		}
	}

	return lastAccepted
}

func (set StorySet) IssueLabels() []string {
	var labels []string

	var hasBugs bool
	var hasFeatures bool
	for _, story := range set {
		if story.Type == "feature" {
			hasFeatures = true
		} else if story.Type == "bug" {
			hasBugs = true
		}
	}

	if hasFeatures {
		labels = append(labels, IssueLabelEnhancement)
	} else if hasBugs {
		labels = append(labels, IssueLabelBug)
	}

	if set.AllAccepted() {
		// everything is accepted; only set labels for types of stories, not status
		return labels
	}

	allUnscheduled := true
	for _, story := range set {
		switch story.State {
		case "accepted":
			// ignore accepted stories; if some are accepted but the rest are
			// unscheduled, it's still unscheduled

		case "unscheduled":
			// only mark if all are unscheduled

		case "started", "finished", "delivered", "rejected":
			// a story is in-progress; report as in-flight
			labels = append(labels, IssueLabelInFlight)
			return labels

		case "unstarted", "planned":
			// something is scheduled
			allUnscheduled = false

		default:
			log.Fatalln("unknown story state:", story.State)
		}
	}

	if allUnscheduled {
		labels = append(labels, IssueLabelUnscheduled)
	} else {
		labels = append(labels, IssueLabelScheduled)
	}

	return labels
}
