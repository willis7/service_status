# AGENTS.md - Coding Agent Guidelines


## MANDATORY: Use td for Task Management

Run td usage --new-session at conversation start (or after /clear). This tells you what to work on next.

Sessions are automatic (based on terminal/agent context). Optional:
- td session "name" to label the current session
- td session --new to force a new session in the same context

Use td usage -q after first read.

## Project Overview

This project runs on a Mac and the project is configured to use Taskfile. When testing the project always use the taskfile.
