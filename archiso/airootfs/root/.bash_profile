# Live ISO root login (tty2 autologin is the rescue console).
[[ -f ~/.bashrc ]] && . ~/.bashrc
cat /etc/motd 2>/dev/null || true
echo "Install: coda-install    Network: iwctl    Session: greetd on tty1"
