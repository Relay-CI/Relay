"""
Relay Demo Automation Script — VS Code Edition
================================================
Everything runs inside VS Code:
  - VS Code integrated terminal for all relay commands
  - Find & Replace edits page.tsx visibly on screen
  - Opens browser to show live preview before and after the change
  - Redeploys, rollbacks, shows secrets, ends on relay --help

Setup:
  pip install pyautogui colorama

Usage:
  1) Start relayd
  2) Open VS Code with C:\\Users\\aloys\\Downloads\\relay\\site already open
  3) Open the integrated terminal (Ctrl+`) — make sure it is focused
  4) Start your screen recorder
  5) Run THIS script from a SEPARATE minimised terminal:
       python relay_demo.py
  6) Click into the VS Code integrated terminal during the countdown

Abort anytime: move mouse to TOP-LEFT corner (pyautogui failsafe)
"""

import os
import sys
import time
import webbrowser
import subprocess
import getpass
from colorama import Fore, Style, init
import pyautogui

init(autoreset=True)

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

TYPING_SPEED       = 0.045
TYPING_SPEED_FAST  = 0.02
PAUSE_SHORT        = 0.8
PAUSE_MEDIUM       = 2.5
PAUSE_LONG         = 5.5
COUNTDOWN_SECONDS  = 5

DEMO_APP_PATH = r"C:\Users\aloys\Downloads\relay\site"

RELAY_URL    = os.environ.get("RELAY_URL",   "")
RELAY_TOKEN  = os.environ.get("RELAY_TOKEN", "")
RELAY_APP    = "site"
RELAY_ENV    = "preview"
RELAY_BRANCH = "main"

# Hardcode your preview URL or leave None to auto-detect from relay status
PREVIEW_URL = os.environ.get("RELAY_PREVIEW_URL", None)

# File to edit (relative to DEMO_APP_PATH)
EDIT_FILE = r"src\app\page.tsx"

# Find & Replace strings — visible on screen during the edit step
ORIGINAL_TITLE = "Push code. Own the box."
UPDATED_TITLE  = "Ship fast. Stay in control."


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def log(msg, color=Fore.CYAN):
    print(f"{color}[demo] {msg}{Style.RESET_ALL}")

def section(title):
    print()
    print(Fore.GREEN + "-" * 56)
    print(f"  {title}")
    print("-" * 56 + Style.RESET_ALL)

def countdown():
    print()
    log(f"Starting in {COUNTDOWN_SECONDS}s — click the VS Code terminal NOW", Fore.YELLOW)
    for i in range(COUNTDOWN_SECONDS, 0, -1):
        print(f"  {Fore.YELLOW}{i}...{Style.RESET_ALL}", end="\r")
        time.sleep(1)
    print()
    log("GO", Fore.GREEN)
    time.sleep(0.4)

def type_cmd(text, speed=None):
    if speed is None:
        speed = TYPING_SPEED if len(text) < 48 else TYPING_SPEED_FAST
    for ch in text:
        pyautogui.typewrite(ch, interval=speed)
        if ch in (" ", "-", "/") and speed == TYPING_SPEED:
            time.sleep(0.03)

def run(cmd, wait=1.2, comment=None):
    """Type a command into the focused terminal and press Enter."""
    if comment:
        log(comment, Fore.WHITE)
    time.sleep(PAUSE_SHORT)
    type_cmd(cmd)
    time.sleep(0.15)
    pyautogui.press("enter")
    time.sleep(wait)

def answer(text, wait=PAUSE_SHORT, comment=None):
    """Answer a wizard prompt."""
    if comment:
        log(comment, Fore.WHITE)
    time.sleep(0.5)
    type_cmd(text)
    time.sleep(0.15)
    pyautogui.press("enter")
    time.sleep(wait)

def just_enter(wait=PAUSE_SHORT, comment=None):
    """Accept a wizard default by pressing Enter only."""
    if comment:
        log(comment, Fore.WHITE)
    time.sleep(0.5)
    pyautogui.press("enter")
    time.sleep(wait)

def pause(seconds, reason=None):
    if reason:
        log(f"Waiting {seconds:.0f}s — {reason}", Fore.MAGENTA)
    time.sleep(seconds)

def cls():
    run("cls", wait=0.5)

def open_browser(url):
    log(f"Opening browser: {url}", Fore.CYAN)
    webbrowser.open(url)

def get_preview_url_from_status():
    """Run relay status out-of-band and pull the first http URL from output."""
    try:
        env = {**os.environ}
        if RELAY_URL:   env["RELAY_URL"]   = RELAY_URL
        if RELAY_TOKEN: env["RELAY_TOKEN"] = RELAY_TOKEN
        result = subprocess.run(
            ["relay", "status",
             "--app", RELAY_APP,
             "--env", RELAY_ENV,
             "--branch", RELAY_BRANCH],
            capture_output=True, text=True, env=env, timeout=10
        )
        for line in result.stdout.splitlines():
            for token in line.split():
                if token.startswith("http"):
                    return token.strip()
    except Exception as e:
        log(f"Could not auto-detect preview URL: {e}", Fore.YELLOW)
    return None

def restore_file_on_disk():
    """
    Silently restore page.tsx to original title after the demo
    so the script is repeatable.
    """
    full_path = os.path.join(DEMO_APP_PATH, EDIT_FILE)
    try:
        with open(full_path, "r", encoding="utf-8") as f:
            content = f.read()
        if UPDATED_TITLE in content:
            content = content.replace(UPDATED_TITLE, ORIGINAL_TITLE, 1)
            with open(full_path, "w", encoding="utf-8") as f:
                f.write(content)
            log("page.tsx restored to original title — ready to run again", Fore.GREEN)
        else:
            log("page.tsx already has original title — nothing to restore", Fore.YELLOW)
    except Exception as e:
        log(f"Could not restore page.tsx: {e}", Fore.YELLOW)


# ---------------------------------------------------------------------------
# VS Code editor step
# ---------------------------------------------------------------------------

def do_code_change():
    """
    1. Open page.tsx from the integrated terminal with `code <file>`
    2. Focus the editor pane
    3. Ctrl+H — Find & Replace, visible to viewer
    4. Type search string, Tab, type replacement, Replace All
    5. Escape, Ctrl+S to save
    6. Ctrl+` back to integrated terminal
    """
    section("Edit page.tsx — Find & Replace")

    # Open the file from the VS Code integrated terminal
    run(f"code {EDIT_FILE}", wait=2.5, comment="Open page.tsx in editor tab")

    # Focus editor pane (Ctrl+1)
    log("Focusing editor pane (Ctrl+1)", Fore.MAGENTA)
    pyautogui.hotkey("ctrl", "1")
    time.sleep(0.8)

    # Open Find & Replace
    log("Opening Find & Replace (Ctrl+H)", Fore.MAGENTA)
    pyautogui.hotkey("ctrl", "h")
    time.sleep(1.2)

    # Type search term
    log(f"Searching for: {ORIGINAL_TITLE}", Fore.WHITE)
    type_cmd(ORIGINAL_TITLE, speed=TYPING_SPEED)
    time.sleep(0.5)

    # Tab to replace field
    pyautogui.press("tab")
    time.sleep(0.5)

    # Type replacement
    log(f"Replacing with: {UPDATED_TITLE}", Fore.WHITE)
    type_cmd(UPDATED_TITLE, speed=TYPING_SPEED)
    time.sleep(0.5)

    # Replace All (Ctrl+Alt+Enter)
    log("Replace All (Ctrl+Alt+Enter)", Fore.WHITE)
    pyautogui.hotkey("ctrl", "alt", "enter")
    time.sleep(0.8)

    # Close Find & Replace bar
    pyautogui.press("escape")
    time.sleep(0.5)

    # Pause so viewer can see the change in the file
    pause(2.5, "viewer reads the change")

    # Save
    log("Saving (Ctrl+S)", Fore.WHITE)
    pyautogui.hotkey("ctrl", "s")
    time.sleep(0.6)

    # Back to integrated terminal — no Alt+Tab needed, just Ctrl+`
    log("Back to integrated terminal (Ctrl+`)", Fore.MAGENTA)
    pyautogui.hotkey("ctrl", "`")
    time.sleep(1.0)


# ---------------------------------------------------------------------------
# Demo sequence
# ---------------------------------------------------------------------------

def run_demo(relay_url, relay_token):
    countdown()

    # ── 0. cd into project (VS Code terminal may open at repo root) ──────────
    section("0 / Navigate to site folder")
    run(f'cd "{DEMO_APP_PATH}"', wait=PAUSE_SHORT, comment="Make sure we are in the site folder")
    run("dir", wait=PAUSE_MEDIUM, comment="Show project files")
    pause(PAUSE_MEDIUM)

    # ── 1. relay init ───────────────────────────────────────────────────────
    # Wizard prompt order:
    #   "Connection type [socket / http]:"  → http
    #   "Server URL (http://...):"          → relay_url
    #   "Token:"                            → relay_token  (shown as ***)
    #   "App name (relay):"                 → Enter
    #   "Env (preview):"                    → Enter
    #   "Branch (main):"                    → Enter
    #   "Save to .relay.json? (yes):"       → Enter
    section("1 / relay init")
    cls()
    run("relay init", wait=PAUSE_SHORT, comment="Interactive setup wizard")
    answer("http",       comment="Select HTTP transport")
    answer(relay_url,    comment="Enter relayd URL")
    answer(relay_token,  comment="Enter token (shown as *** in terminal)")
    just_enter(          comment="Accept app name default")
    just_enter(          comment="Accept env default")
    just_enter(          comment="Accept branch default")
    just_enter(wait=PAUSE_MEDIUM, comment="Save .relay.json")
    pause(PAUSE_SHORT)
    run("type .relay.json", wait=PAUSE_MEDIUM, comment="Show generated config")
    pause(PAUSE_MEDIUM)

    # ── 2. relay list — nothing deployed yet ────────────────────────────────
    section("2 / relay list — empty history")
    cls()
    run("relay list --app site", wait=PAUSE_MEDIUM, comment="No deploys yet")
    pause(PAUSE_MEDIUM)

    # ── 3. First deploy ─────────────────────────────────────────────────────
    section("3 / First deploy")
    cls()
    run("relay deploy --stream", wait=PAUSE_LONG * 2, comment="Deploy — streams build output live")
    pause(PAUSE_LONG, "build output settling")

    # ── 4. relay list — deploy appears ──────────────────────────────────────
    section("4 / relay list — deploy in history")
    run("relay list --app site", wait=PAUSE_MEDIUM, comment="Confirm deploy recorded")
    pause(PAUSE_MEDIUM)

    # ── 5. Open preview in browser ──────────────────────────────────────────
    section("5 / Open preview URL — before change")
    preview_url = PREVIEW_URL or get_preview_url_from_status()
    if preview_url:
        open_browser(preview_url)
        pause(PAUSE_LONG, "browser loading — viewer sees current title")
    else:
        log("No preview URL found — set RELAY_PREVIEW_URL or hardcode PREVIEW_URL", Fore.YELLOW)
        pause(PAUSE_SHORT)

    # ── 6. Edit page.tsx in VS Code ─────────────────────────────────────────
    do_code_change()

    # ── 7. Redeploy — delta only ────────────────────────────────────────────
    section("7 / Redeploy — changed files only")
    cls()
    run("relay deploy --stream", wait=PAUSE_LONG * 2, comment="Second deploy — faster, delta only")
    pause(PAUSE_LONG, "delta deploy settling")

    # ── 8. Browser — show updated title ─────────────────────────────────────
    section("8 / Preview — title has changed")
    if preview_url:
        open_browser(preview_url)
        pause(PAUSE_LONG, "viewer sees updated title in browser")

    # ── 9. Rollback ─────────────────────────────────────────────────────────
    section("9 / Rollback")
    cls()
    run(
        "relay rollback --app site --env preview --branch main",
        wait=PAUSE_LONG,
        comment="One command — previous image restored instantly"
    )
    pause(PAUSE_LONG)

    # ── 10. Secrets ─────────────────────────────────────────────────────────
    section("10 / Secrets")
    run(
        "relay secrets list --app site --env preview",
        wait=PAUSE_MEDIUM,
        comment="Secrets management"
    )
    pause(PAUSE_MEDIUM)

    # ── 11. End card ────────────────────────────────────────────────────────
    section("11 / End card")
    cls()
    run("relay --help", wait=PAUSE_MEDIUM, comment="Full command overview")
    pause(PAUSE_LONG)

    # ── Restore file so demo is repeatable ──────────────────────────────────
    restore_file_on_disk()

    print()
    log("Demo complete — stop your recording!", Fore.GREEN)


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------

def check_safety():
    pyautogui.FAILSAFE = True
    print()
    print(Fore.RED + "  SAFETY: Move mouse to TOP-LEFT corner at any time to abort.")
    print(Fore.RED + "  VS Code must be open with the integrated terminal focused." + Style.RESET_ALL)
    print()

if __name__ == "__main__":
    check_safety()

    print(Fore.CYAN + """
  +------------------------------------------+
  |   Relay Demo Script — VS Code Edition    |
  +------------------------------------------+
""" + Style.RESET_ALL)

    print(f"  DEMO_APP_PATH  = {DEMO_APP_PATH}")
    print(f"  APP/ENV/BRANCH = {RELAY_APP}/{RELAY_ENV}/{RELAY_BRANCH}")
    print(f"  EDIT_FILE      = {EDIT_FILE}")
    print(f"  PREVIEW_URL    = {PREVIEW_URL or '(auto-detect from relay status)'}")
    print()

    if not RELAY_URL:
        RELAY_URL = input(f"  {Fore.YELLOW}relayd URL{Style.RESET_ALL} [http://127.0.0.1:8080]: ").strip()
        if not RELAY_URL:
            RELAY_URL = "http://127.0.0.1:8080"

    if not RELAY_TOKEN:
        RELAY_TOKEN = getpass.getpass(
            f"  {Fore.YELLOW}relayd token{Style.RESET_ALL} (hidden, leave blank if none): "
        ).strip()

    print(f"\n  RELAY_URL   = {RELAY_URL}")
    print(f"  RELAY_TOKEN = {'(set)' if RELAY_TOKEN else '(none)'}")
    print()

    try:
        run_demo(RELAY_URL, RELAY_TOKEN)
    except pyautogui.FailSafeException:
        print()
        log("Aborted by failsafe.", Fore.RED)
        sys.exit(0)
    except KeyboardInterrupt:
        print()
        log("Aborted by Ctrl+C.", Fore.RED)
        sys.exit(0)