# eReminders

eReminders is a reminders program that has __no user interface__. You create reminder text files in a watched directory, and eReminders will automatically email you when the reminder's date is elapsed. 

This project works really great for those who follow the "Inbox Zero" philosophy.

## Usage

You create reminders using a very simple text file format like this:
```
2019-02-17 13:00

Upload eReminders Source code
```

Where the first line is a fuzzy, human date format (one that can be parsed by Oleg Lebedev's [when library](https://github.com/olebedev/when)). The second line is the title of the reminder. That's it.

Then, start an instance of ereminders and tell it which directory to watch for these files.

```
$ ereminders -d ~/reminders
```

eReminders will watch this directory for changes. If you delete a reminder, create a new one, or edit an existing one, the eReminders daemon will know about it. 

When you create a reminder, it will be scheduled to email you with the reminder title as the subject of the email. If you want to take a look at all the currently scheduled reminders (and make sure eReminders parsed the date correctly), you can run `ereminders -l`. 

eReminders also supports recurring reminders. This is really helpful for remembering to pay bills or something. To create a recurring reminder, edit the first line of your reminder to declare whether or not its a `repeat daily`, `repeat weekly`, or `repeat monthly` reminder, like this:

```
repeat monthly
13:37

Pay bills
```

If eReminders encountered an error parsing your reminder file, it will rename it to `filename.ERROR`.


## Configuration

eReminders requires a configuration file so it knows how to email you. By default, eReminders looks in `~/.config/ereminders/config.toml`. Copy over and edit the included `sample_config.toml` for a quick way to get started. 

