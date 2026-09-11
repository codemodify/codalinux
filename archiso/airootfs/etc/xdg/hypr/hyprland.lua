-- CodaLinux Hyprland stub (Lua / Hyprland 0.55+).
-- Legacy hyprland.conf is gone — Hyprland 0.57 drops that format.
-- Docs: https://wiki.hypr.land/Configuring/Start/

local terminal = "foot"
local fileManager = "thunar"
local browser = "firefox"
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
-- VirtualBox VMSVGA / software-render fallbacks (coda-hyprland also sets these).
hl.env("WLR_NO_HARDWARE_CURSORS", "1")
hl.env("WLR_RENDERER_ALLOW_SOFTWARE", "1")

hl.on("hyprland.start", function()
    hl.exec_cmd("dbus-update-activation-environment --systemd WAYLAND_DISPLAY XDG_CURRENT_DESKTOP")
    hl.exec_cmd("systemctl --user start pipewire pipewire-pulse wireplumber")
    hl.exec_cmd("/usr/lib/xdg-desktop-portal-hyprland")
    hl.exec_cmd("hyprpaper")
    hl.exec_cmd("hypridle")
    -- TODO: hl.exec_cmd("ags run /usr/local/share/codalinux/ags") once desktop/ags/ is implemented
end)

hl.config({
    general = {
        gaps_in     = 6,
        gaps_out    = 12,
        border_size = 2,
        -- Theme placeholder — see branding/themes/README.md
        col = {
            active_border   = { colors = { "rgba(7fb4c8ee)" } },
            inactive_border = "rgba(595959aa)",
        },
    },

    decoration = {
        rounding = 8,
    },

    animations = {
        enabled = false,
    },

    input = {
        kb_layout    = "us",
        follow_mouse = 1,
        touchpad = {
            natural_scroll = true,
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
hl.bind(mainMod .. " + D", hl.dsp.exec_cmd(terminal))
-- TODO: SUPER + D → AGS launcher

hl.bind(mainMod .. " + mouse:272", hl.dsp.window.drag(), { mouse = true })
hl.bind(mainMod .. " + mouse:273", hl.dsp.window.resize(), { mouse = true })
