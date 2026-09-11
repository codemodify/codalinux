-- CodaLinux Hyprland session (Lua / Hyprland 0.55+).
-- Unified shell is vendored AGS/Astal (coda-ags). Waybar is not used.
-- Docs: https://wiki.hypr.land/Configuring/Start/

local terminal = "foot"
local fileManager = "thunar"
local browser = "firefox"
local launcher = "coda-ags toggle launcher"
local settings = "coda-ags toggle control-center"
local clipboard = "coda-ags toggle clipboard"
local mainMod = "SUPER"

hl.monitor({
    output   = "",
    mode     = "preferred",
    position = "auto",
    scale    = 1,
})

hl.env("XCURSOR_SIZE", "24")
hl.env("XDG_CURRENT_DESKTOP", "Hyprland")
hl.env("XDG_SESSION_TYPE", "wayland")
hl.env("XDG_SESSION_DESKTOP", "Hyprland")
hl.env("GI_TYPELIB_PATH", "/usr/local/lib/girepository-1.0")
hl.env("LD_LIBRARY_PATH", "/usr/local/lib")
hl.env("XDG_DATA_DIRS", "/usr/local/share:/usr/share")
-- VirtualBox VMSVGA / software-render fallbacks (coda-hyprland also sets these).
hl.env("WLR_NO_HARDWARE_CURSORS", "1")
hl.env("WLR_RENDERER_ALLOW_SOFTWARE", "1")

hl.on("hyprland.start", function()
    hl.exec_cmd("dbus-update-activation-environment --systemd WAYLAND_DISPLAY XDG_CURRENT_DESKTOP XDG_SESSION_TYPE")
    hl.exec_cmd("systemctl --user start pipewire pipewire-pulse wireplumber")
    hl.exec_cmd("/usr/lib/xdg-desktop-portal-hyprland")
    hl.exec_cmd("/usr/lib/polkit-gnome/polkit-gnome-authentication-agent-1")
    hl.exec_cmd("hyprpaper -c /etc/xdg/hypr/hyprpaper.conf")
    hl.exec_cmd("coda-ags")
    hl.exec_cmd("hypridle")
    hl.exec_cmd("blueman-applet")
    hl.exec_cmd("wl-paste --type text --watch cliphist store")
end)

hl.config({
    general = {
        gaps_in     = 6,
        gaps_out    = 12,
        border_size = 4,
        col = {
            active_border   = { colors = { "rgba(3dd6f5ff)" } },
            inactive_border = "rgba(3a4550ff)",
        },
    },

    decoration = {
        rounding = 8,
        -- Keep chrome readable on floating/overlapping windows.
        -- Blur stays off: VirtualBox software-render (pixman) is the live default.
        shadow = {
            enabled = true,
            range   = 12,
            render_power = 3,
            color   = "rgba(00000066)",
        },
        blur = {
            enabled = false,
        },
    },

    animations = {
        enabled = false,
    },

    input = {
        kb_layout    = "us",
        follow_mouse = 1,
        touchpad = {
            natural_scroll = true,
            tap_to_click   = true,
        },
    },

    misc = {
        disable_hyprland_logo     = true,
        disable_splash_rendering  = true,
        force_default_wallpaper   = 0,
    },

    cursor = {
        no_hardware_cursors = true,
    },
})

hl.bind(mainMod .. " + Return", hl.dsp.exec_cmd(terminal))
hl.bind(mainMod .. " + Q", hl.dsp.window.close())
hl.bind(mainMod .. " + E", hl.dsp.exec_cmd(fileManager))
hl.bind(mainMod .. " + F", hl.dsp.exec_cmd(browser))
hl.bind(mainMod .. " + L", hl.dsp.exec_cmd("hyprlock"))
hl.bind(mainMod .. " + D", hl.dsp.exec_cmd(launcher))
hl.bind(mainMod .. " + SPACE", hl.dsp.exec_cmd(launcher))
hl.bind(mainMod .. " + comma", hl.dsp.exec_cmd(settings))
hl.bind(mainMod .. " + V", hl.dsp.exec_cmd(clipboard))
hl.bind(mainMod .. " + N", hl.dsp.exec_cmd("coda-hypr-ws add"))
hl.bind(mainMod .. " + equal", hl.dsp.exec_cmd("coda-hypr-ws add"))
hl.bind(mainMod .. " + SHIFT + equal", hl.dsp.exec_cmd("coda-hypr-ws add"))
hl.bind(mainMod .. " + plus", hl.dsp.exec_cmd("coda-hypr-ws add"))
hl.bind(mainMod .. " + minus", hl.dsp.exec_cmd("coda-hypr-ws remove"))
hl.bind(mainMod .. " + T", hl.dsp.exec_cmd("coda-hypr-ws toggle-stack"))
hl.bind(mainMod .. " + TAB", hl.dsp.window.cycle_next())
hl.bind(mainMod .. " + SHIFT + TAB", hl.dsp.window.cycle_next({ next = false }))

-- New windows on a stacked (floating) workspace join the overlap cascade.
hl.on("window.open", function()
    hl.exec_cmd("coda-hypr-ws apply-new")
end)

for i = 1, 10 do
    local key = i % 10
    hl.bind(mainMod .. " + " .. key, hl.dsp.focus({ workspace = i }))
    hl.bind(mainMod .. " + SHIFT + " .. key, hl.dsp.window.move({ workspace = i }))
end

hl.bind(mainMod .. " + mouse:272", hl.dsp.window.drag(), { mouse = true })
hl.bind(mainMod .. " + mouse:273", hl.dsp.window.resize(), { mouse = true })

hl.bind("XF86AudioRaiseVolume", hl.dsp.exec_cmd("wpctl set-volume -l 1 @DEFAULT_AUDIO_SINK@ 5%+"), { locked = true, repeating = true })
hl.bind("XF86AudioLowerVolume", hl.dsp.exec_cmd("wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-"), { locked = true, repeating = true })
hl.bind("XF86AudioMute", hl.dsp.exec_cmd("wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle"), { locked = true })
hl.bind("XF86AudioMicMute", hl.dsp.exec_cmd("wpctl set-mute @DEFAULT_AUDIO_SOURCE@ toggle"), { locked = true })
hl.bind("XF86MonBrightnessUp", hl.dsp.exec_cmd("brightnessctl -e4 -n2 set 5%+"), { locked = true, repeating = true })
hl.bind("XF86MonBrightnessDown", hl.dsp.exec_cmd("brightnessctl -e4 -n2 set 5%-"), { locked = true, repeating = true })
hl.bind("XF86AudioNext", hl.dsp.exec_cmd("playerctl next"), { locked = true })
hl.bind("XF86AudioPause", hl.dsp.exec_cmd("playerctl play-pause"), { locked = true })
hl.bind("XF86AudioPlay", hl.dsp.exec_cmd("playerctl play-pause"), { locked = true })
hl.bind("XF86AudioPrev", hl.dsp.exec_cmd("playerctl previous"), { locked = true })
