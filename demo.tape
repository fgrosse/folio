# A short tour of folio on a made-up account. Record it with: vhs demo.tape
#
# The seed makes it the same account every time. The prices are the ones of the day, so the values
# differ from one recording to the next.

Output demo.gif

Require folio

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Width 1200
Set Height 640
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms

Type "folio demo --seed 7"
Sleep 500ms
Enter
Sleep 3s

# Holdings: walk down the lots
Down@400ms 3
Sleep 1s

# The details of the selected lot, which follow the selection: its gain and the tax on it
Enter
Sleep 2.5s
Up@700ms 3
Sleep 2s
Down@400ms 3
Escape
Sleep 1s

# Sell some shares of the selected lot
Type "s"
Sleep 1s
Type "5"
Tab
Type "240"
Tab 2
Type "Paying for the new bike"
Sleep 1.5s
Ctrl+S
Sleep 2s

# Vesting: release the vest that is due
Type "2"
Sleep 2s
Type "r"
Sleep 1s
# The dialog opens with the shares that vested: 44 of the 73 arrived, the rest was withheld for tax
Backspace 2
Type "44 @238.90"
Sleep 1s
Enter
Sleep 2s

# Grants
Type "3"
Sleep 2.5s

# Sales: the one just made, with its note
Type "4"
Sleep 1.5s
Down@400ms
Sleep 2s

# Back to the holdings, which have the released shares now
Type "1"
Sleep 3s

Type "q"
Sleep 500ms
