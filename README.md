# Organizer bot

<a href="https://top.gg/bot/897597207086247956">
  <img src="https://top.gg/api/widget/897597207086247956.svg">
</a>

<hr>

## About

This bot can be used to organize work and tasks on your Discord channel.

Currently, supported features are:
- to-do lists
- reminders

## Usage

### Configuration
`/organizer config timezone` - get the currently set timezone
`/organizer config timezone <timezone_name>` - set the timezone

### To-do lists

`/organizer todo add <text>` - Add a new task to the channel to-do list
`/organizer todo show` - Show all current tasks
`/organizer todo done` - Mark a task as done

### Reminders

Reminders can be used to send reminders on a channel at a date.

> You must have a timezone set on the channel to make the reminders work!

`/organizer reminder add date: <date> text: <text>` - Add a new reminder to the channel to-do list
`/organizer reminder show` - Show all reminders
`/organizer reminder remove` - Remove a reminder
