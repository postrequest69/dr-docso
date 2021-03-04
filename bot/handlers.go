package bot

import (
	"fmt"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/post04/dr-docso/docs"
	"github.com/post04/dr-docso/glob"
)

// DocsHelpEmbed - the embed to give help to the docs command.
var DocsHelpEmbed = &discordgo.MessageEmbed{
	Title: "Docs help!",
}

// HandleDoc  is the handler for the doc command.
func HandleDoc(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string, prefix string) {
	var msg *discordgo.MessageEmbed
	fields := strings.Fields(m.Content)
	switch len(fields) {
	case 0: // probably impossible
		return
	case 1: // only the invocation
		msg = helpShortResponse()
	case 2: // invocation + arg
		msg = pkgResponse(fields[1])
	case 3: // invocation + pkg + func
		if strings.Contains(fields[2], ".") {
			split := strings.Split(fields[2], ".")
			msg = methodResponse(fields[1], split[0], split[1])
		} else {
			msg = queryResponse(fields[1], fields[2])
		}
	default:
		msg = errResponse("Too many arguments.")
	}

	if msg == nil {
		msg = errResponse("No results found, possibly an internal error.")
	}
	s.ChannelMessageSendEmbed(m.ChannelID, msg)
}

func queryResponse(pkg, name string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occurred while fetching the page for pkg `%s`", pkg)
	}

	var msg string
	for _, fn := range doc.Functions {
		if fn.Type == docs.FnNormal && strings.EqualFold(fn.Name, name) {
			// match found
			name = fn.Name
			msg += fmt.Sprintf("`%s`", fn.Signature)
			if len(fn.Comments) > 0 {
				msg += fmt.Sprintf("\n%s", fn.Comments[0])
			} else {
				msg += "\n*no information*"
			}
			if fn.Example != "" {
				msg += fmt.Sprintf("\n\nExample:\n```go\n%s\n```", fn.Example)
			}
		}
	}

	if msg == "" {
		for _, t := range doc.Types {
			if strings.EqualFold(name, t.Name) {
				msg += fmt.Sprintf("```go\n%s\n```\n", t.Signature)
				if len(t.Comments) > 0 {
					msg += t.Comments[0]
				} else {
					msg += "*no information available*\n"
				}
			}
		}
	}

	if msg == "" {
		return errResponse("No type or function `%s` found in package `%s`", name, pkg)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message was trimmed to fit the 2k character limit*", msg[:1950])
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s: %s", pkg, name),
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%v#%v", doc.URL, name),
		},
	}
}

func errResponse(format string, args ...interface{}) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Error",
		Description: fmt.Sprintf(format, args...),
	}
}

func typeResponse(pkg, name string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occurred while getting the page for the package `%s`", pkg)
	}
	if len(doc.Types) == 0 {
		return errResponse("Package `%s` seems to have no type definitions", pkg)
	}

	var msg string

	for _, t := range doc.Types {
		if strings.EqualFold(t.Name, name) {
			// got a match

			// To get the hyper link (case it's case sensitive)
			name = t.Name
			msg += fmt.Sprintf("```go\n%s\n```", t.Signature)
			if len(t.Comments) > 0 {
				msg += fmt.Sprintf("\n%s", t.Comments[0])
			} else {
				msg += "\n*no information*"
			}
		}
	}

	if msg == "" {
		return errResponse("Package `%s` does not have type `%s`", pkg, name)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message is trimmed to fit the 2k character limit*", msg[:1950])
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s: type %s", pkg, name),
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%v#%v", doc.URL, name),
		},
	}
}

func helpShortResponse() *discordgo.MessageEmbed {
	return DocsHelpEmbed
}

func pkgResponse(pkg string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occured when requesting the page for the package `%s`", pkg)
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Info for %s", pkg),
		Description: fmt.Sprintf("Types: %v\nFunctions:%v", len(doc.Types), len(doc.Functions)),
	}
	return embed
}

func methodResponse(pkg, t, name string) *discordgo.MessageEmbed {
	if strings.Contains(t, "*") ||
		strings.Contains(name, "*") {
		return methodGlobResponse(pkg, t, name)
	}

	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("Error while getting the page for the package `%s`", pkg)
	}
	if len(doc.Functions) == 0 {
		return errResponse("Package `%s` seems to have no functions", pkg)
	}

	var msg string
	var hyper string
	for _, fn := range doc.Functions {
		if fn.Type == docs.FnMethod &&
			strings.EqualFold(fn.Name, name) &&
			strings.EqualFold(fn.MethodOf, t) {
			hyper = fmt.Sprintf("%v#%v.%v", doc.URL, fn.MethodOf, fn.Name)
			msg += fmt.Sprintf("`%s`", fn.Signature)
			if len(fn.Comments) > 0 {
				msg += fmt.Sprintf("\n%s", fn.Comments[0])
			} else {
				msg += "\n*no info*"
			}
			if fn.Example != "" {
				msg += fmt.Sprintf("\nExample:\n```\n%s\n```", fn.Example)
			}
		}
	}
	if msg == "" {
		return errResponse("Package `%s` does not have `func(%s) %s`", pkg, t, name)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message is trimmed to fit the 2k character limit*", msg[:1950])
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s: func(%s) %s", pkg, t, name),
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: hyper,
		},
	}
}

// PagesShortResponse - basically just a help command for the pages system :p
func PagesShortResponse(state, prefix string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Help %v", state),
		Description: fmt.Sprintf("It seems you didn't have enough arguments, so here's an example!\n\n%v%v strings", prefix, state),
	}
}

// FuncsPages - for the reaction pages with all the functions in a package!
func FuncsPages(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string, prefix string) {
	fields := strings.Fields(m.Content)
	switch len(fields) {
	case 0: // probably impossible
		return
	case 1: // send a help command here
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("getfuncs", prefix))
		return
	case 2: // command + pkg (send page if possible)
		//TODO impl this
		doc, err := getDoc(fields[1])
		if err != nil || doc == nil {
			s.ChannelMessageSendEmbed(m.ChannelID, errResponse("Error while getting the page for the package `%s`", fields[1]))
			return
		}
		var pageLimit = int(math.Ceil(float64(len(doc.Functions)) / 10.0))
		var page = &ReactionListener{
			Type:        "functions",
			CurrentPage: 1,
			PageLimit:   pageLimit,
			UserID:      m.Author.ID,
			Data:        doc,
			LastUsed:    MakeTimestamp(),
		}
		textTosend := formatForMessage(page)
		m, err := s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "functions",
			Description: textTosend,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Page 1/" + fmt.Sprint(pageLimit),
			},
		})
		if err != nil {
			return
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, leftArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, rightArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, destroyEmoji)
		pageListeners[m.ID] = page
		return
	default: // too many arguments
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("getfuncs", prefix))
		return
	}

}

// TypesPages - for the reaction pages with all the types in a package!
func TypesPages(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string, prefix string) {
	fields := strings.Fields(m.Content)
	switch len(fields) {
	case 0: // probably impossible
		return
	case 1: // send a help command here
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("gettypes", prefix))
		return
	case 2: // command + pkg (send page if possible)
		//TODO impl this
		doc, err := getDoc(fields[1])
		if err != nil || doc == nil {
			s.ChannelMessageSendEmbed(m.ChannelID, errResponse("Error while getting the page for the package `%s`", fields[1]))
			return
		}
		var pageLimit = int(math.Ceil(float64(len(doc.Types)) / 10.0))
		var page = &ReactionListener{
			Type:        "types",
			CurrentPage: 1,
			PageLimit:   pageLimit,
			UserID:      m.Author.ID,
			Data:        doc,
			LastUsed:    MakeTimestamp(),
		}
		textTosend := formatForMessage(page)
		m, err := s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "types",
			Description: textTosend,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Page 1/" + fmt.Sprint(pageLimit),
			},
		})
		if err != nil {
			return
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, leftArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, rightArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, destroyEmoji)
		pageListeners[m.ID] = page
		return
	default: // too many arguments
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("gettypes", prefix))
		return
	}
}

func methodGlobResponse(pkg, t, name string) *discordgo.MessageEmbed {
	reT, err := glob.Compile(t)
	if err != nil {
		return errResponse("Error processing glob pattern:\n```\n%s\n```", err)
	}
	reN, err := glob.Compile(name)
	if err != nil {
		return errResponse("Error processing glob pattern:\n```\n%s\n```", err)
	}
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occurred while getting the page for the package `%s`", pkg)
	}

	if len(doc.Functions) == 0 || len(doc.Types) == 0 {
		return errResponse("No results found matching the expression `%s.%s` in package `%s`", t, name, pkg)
	}

	var msg string
	for _, fn := range doc.Functions {
		if fn.Type == docs.FnMethod &&
			reT.MatchString(fn.MethodOf) &&
			reN.MatchString(fn.Name) {
			msg += fmt.Sprintf("`%s`:\n", fn.Signature)
			if len(fn.Comments) > 0 {
				msg += fn.Comments[0]
			} else {
				msg += "*no information available*"
			}
		}
	}
	if msg == "" {
		return errResponse("No results found matching the expression `%s.%s` in package `%s`", t, name, pkg)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message was trimmed to fit the 2k character limit*", msg[:1950])
	}
	return &discordgo.MessageEmbed{
		Title:       "Matches",
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: doc.URL,
		},
	}
}
