# EchoBoard 📣

> An open-source, multi-user social media management, content planning, and analytics platform designed for complete data ownership.

EchoBoard is a comprehensive tool for planning, tracking, measuring, and optimizing your blog and social media content. Built to be self-hosted on a basic Virtual Private Server (VPS) or run locally, it gives creators, agencies, and brands complete control over their content workflow, audience interactions, and analytics without recurring SaaS fees.

---

## 🚀 Features

### 1. 📅 Content Calendar
Visualize your content strategy with precision.
* **Exact Timing:** See exactly what has been posted and what is scheduled down to the minute.
* **Flexible Views:** Toggle between daily, weekly, monthly, and custom adjustable timeframes.
* **Drag-and-Drop:** Easily reschedule planned content.

### 2. ✅ Content Approval & Workflow System
A unified space to create, review, and finalize content.
* **Creator Interface:** Upload media (photos/videos), draft captions, select target platforms, and propose posting dates.
* **Content Search:** Quickly find past or drafted content using tags, keywords, or platform filters.
* **Viewer/Editor:** A collaborative interface for team members to review, leave feedback, edit, and approve content for scheduling.

### 3. 💬 Interactions & Unified Inbox
Never miss a conversation. This section aggregates all incoming communications into a familiar chat-like interface.
* **Social Comments:** View and reply to comments on your posts across all connected platforms.
* **Direct Messages & Emails:** Handle DMs, email replies, and live website chats in one continuous conversation thread.
* **Real-time Sync:** Conversations update as interactions happen.

### 4. 👥 People (Integrated CRM)
Consolidate your audience data. 
* **Unified Profiles:** Combine user contact information and cross-platform conversation history into a single, comprehensive information card.
* **Global Search:** Search for users across all connected platforms.
* **Relationship Tracking:** Monitor brand loyalty, interaction frequency, and past support queries.

### 5. 📊 Tracking, Reporting & Analytics
Own your metrics with built-in, comprehensive data analysis.
* **Custom Reports:** Generate highly detailed reports on engagement, reach, and conversion.
* **Built-in Link & Page Tracking:** Where native platform tracking falls short, EchoBoard provides custom tracking links and pixel-like usage tracking to measure true click-throughs and conversions.

### 6. 🔌 Integrations
Seamlessly connect with major platforms via their official APIs.
* **Social:** Meta (Facebook & Instagram), TikTok, YouTube, Snapchat, X/Twitter.
* **E-Commerce:** Shopify (track social-to-sales pipelines).
* **Audio/Music:** Spotify (integration spanning posting, sales, and reporting).

---

## 🏗️ Architecture & Technology Decisions

To ensure EchoBoard is performant, secure, and easy to maintain on a basic VPS, the following architectural decisions have been established:

### Core Technology Stack
* **Backend:** **Go (Golang)**. Go is selected for its incredibly high concurrency performance, which is essential for handling multiple incoming webhooks (social media comments, DMs) and scheduled posting routines simultaneously. It compiles to a single binary, making VPS deployment virtually effortless and highly resource-efficient.
* **Frontend:** **SvelteKit** with **Tailwind CSS**. SvelteKit provides a highly reactive, fast user experience necessary for complex UIs like calendars and real-time chat interfaces, all while outputting optimized, lightweight client code.
* **API:** RESTful architecture with WebSockets for real-time chat/interaction updates.

### Deployment: Local-First & VPS Optimized
* **Architecture:** The system is designed to be **local-first** for solo developers or testers, with a seamless path to VPS deployment for multi-user teams. 
* By packaging the Go backend and the compiled Svelte frontend together, you can run the entire application via a single executable without configuring heavy web servers.

### Database Strategy: Self-Hosted Data Ownership
* **Primary Database:** **PostgreSQL (Self-hosted)**. A self-hosted PostgreSQL instance running on your VPS ensures you maintain 100% ownership of your data, avoiding recurring cloud database fees.
* **Development/Local Mode:** **SQLite**. For local execution or single-user instances, the Go backend will support a SQLite adapter, allowing the system to run entirely out of a local file without any database installation.

### Security & Safety Considerations
* **Authentication:** OAuth2 for connecting third-party platforms (ensuring we never store user social passwords). 
* **Data Encryption:** API keys and integration tokens for Meta, Shopify, etc., are encrypted at rest in the database.
* **Webhooks:** Strict validation and signature verification for all incoming webhooks to prevent payload spoofing.
* **Sanitization:** Aggressive input sanitization on the Interactions/Reactions module to prevent Cross-Site Scripting (XSS) from malicious social media comments.

### User Creation Workflow
* **Admin-Bootstrap:** On the very first run, the system prompts the terminal/console to create the master Admin account.
* **Invite-Only:** To protect VPS instances from unauthorized public registration, self-service account creation is disabled by default. The Admin must generate secure, time-limited invite links or manually provision team members.

### Client Strategy: Web vs. Mobile
* **Phase 1: Progressive Web App (PWA).** Building a deeply responsive, browser-based PWA is the priority. This provides an app-like experience on both desktop and mobile devices immediately, bypassing app store approval bottlenecks and maintaining a unified codebase.
* **Phase 2: Native Apps.** Staggering the launch allows the API and core features to mature. Native iOS/Android apps can be introduced later if specific device capabilities (like aggressive background notifications or raw media rendering) become strictly necessary.

---

## 🛠️ Getting Started

*(Instructions placeholder for post-release)*

```bash
# Clone the repository
git clone [https://github.com/yourusername/echoboard.git](https://github.com/yourusername/echoboard.git)

# Navigate to backend and build the Go binary
cd echoboard/backend
go build -o echoboard main.go

# Run the initialization setup
./echoboard --setup
