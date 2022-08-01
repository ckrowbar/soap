package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
	Guild string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Guild, "g", "", "Guild ID")
	flag.Parse()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal("error creating Discord session,", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsAll

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
		return
	}

	soap(dg)
	fmt.Println("Done.")

	// Cleanly close down the Discord session.
	dg.Close()
}

func soap(dg *discordgo.Session) {
	// Scrape all messages in the past 30 days and store them in messsages
	timeThreshold := time.Now().UTC().AddDate(0, -1, -1)
	messages := make([]discordgo.Message, 0)

	channels, err := dg.GuildChannels(Guild)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Scraping messages...")
	for _, channel := range channels {
		scrapeChannelMessages(dg, channel.ID, &messages, timeThreshold, "")
	}

	// Go through each message and update sent map
	fmt.Println("Updating sent map...")
	sent := make(map[string]bool)
	for _, m := range messages {
		if sent[m.Author.ID] {
			continue
		}
		sent[m.Author.ID] = true
	}

	// Kick all members that have not sent a message in the past month
	fmt.Println("Kicking members...")
	kickList := kickList(dg, Guild, sent)

	for _, u := range kickList {
		fmt.Println("Kicking User: ", u.Username)
		kickMember(dg, u.ID)
	}
}

func scrapeChannelMessages(dg *discordgo.Session, channelID string, messages *[]discordgo.Message, timeThreshold time.Time, before string) {
	// Scrape the past 100 messages in the channel before the provided ID and store them in messages
	msgs, err := dg.ChannelMessages(channelID, 100, before, "", "")

	if err != nil {
		log.Fatal(err)
	}

	// Add all messages to program wide message log bank
	// TODO: Optimize DS to only do a single dereference + append (parity between msgs and messages in DS)
	for _, m := range msgs {
		*messages = append(*messages, *m)
	}

	// If returned msgs < 100, there are no more messages to be scraped
	if len(msgs) < 100 {
		return
	}

	// If the last message is not before the time threshold, then we should continue scraping
	if msgs[99].Timestamp.After(timeThreshold) {
		scrapeChannelMessages(dg, channelID, messages, timeThreshold, msgs[99].ID)
	}
}

func kickList(dg *discordgo.Session, channelID string, sent map[string]bool) []discordgo.User {
	// Return all members in the channel that have not sent a message in the past month
	dg.RequestGuildMembers(Guild, "", 0, "", false)
	users, _ := dg.GuildMembers(Guild, "", 1000)

	kickList := make([]discordgo.User, 0)
	for _, u := range users {
		if sent[u.User.ID] || u.User.Bot {
			continue
		}

		kickList = append(kickList, *u.User)
	}

	return kickList
}

func kickMember(dg *discordgo.Session, memberID string) {
	// Kicks the member with the provided ID and sends them a DM with invite link
	channel, err := dg.UserChannelCreate(memberID)
	if err != nil {
		log.Fatal(err)
		return
	}
	invite, err := dg.ChannelInviteCreate("857620594424807464", discordgo.Invite{
		MaxAge:    172800,
		Temporary: false,
	})
	if err != nil {
		log.Fatal(err)
	}
	dg.ChannelMessageSend(channel.ID, "Hi, this is an automated message from `Nyanpasu` to notify you that you've been kicked due inactivity on the server this past month. Note that this filtering system is automatic as per Nyanpasu's rules and is not personal in any way, shape or form. If you'd like to rejoin, you can do so by clicking the link below. If this link expires, feel free to ask for a new invite from another member :) \n\nhttps://discord.gg/"+invite.Code)
	time.Sleep(time.Second * 4)
	dg.GuildMemberDelete(Guild, memberID)
}
