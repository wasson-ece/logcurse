(function () {
    "use strict";

    const CHUNK_SIZE = 200;
    const GAP_CHUNK = 250;

    // Directory mode: read ?file= param from URL
    const urlParams = new URLSearchParams(window.location.search);
    const fileParam = urlParams.get("file");
    const fileSuffix = fileParam ? "&file=" + encodeURIComponent(fileParam) : "";
    const fileQuery = fileParam ? "?file=" + encodeURIComponent(fileParam) : "";

    let totalLines = 0;
    let loadedRanges = []; // [{start, end}]
    let comments = [];
    let loading = false;
    let selectedRange = null; // {start, end}
    let rwMode = false;

    const fileContent = document.getElementById("file-content");
    const commentContent = document.getElementById("comment-content");
    const fileInfo = document.getElementById("file-info");

    async function fetchLines(start, end) {
        const resp = await fetch(`/api/lines?start=${start}&end=${end}${fileSuffix}`);
        return resp.json();
    }

    async function fetchComments() {
        const resp = await fetch("/api/comments" + fileQuery);
        return resp.json();
    }

    async function fetchConfig() {
        const resp = await fetch("/api/config");
        return resp.json();
    }

    function getAuthor() {
        return localStorage.getItem("logcurse_author") || "";
    }

    function promptAuthor() {
        let author = getAuthor();
        if (!author) {
            author = prompt("Enter your name (used for comment IDs):");
            if (author) {
                localStorage.setItem("logcurse_author", author.trim());
                return author.trim();
            }
            return "";
        }
        return author;
    }

    function isRangeLoaded(start, end) {
        return loadedRanges.some(r => r.start <= start && r.end >= end);
    }

    function mergeRange(start, end) {
        loadedRanges.push({ start, end });
        loadedRanges.sort((a, b) => a.start - b.start);
        const merged = [loadedRanges[0]];
        for (let i = 1; i < loadedRanges.length; i++) {
            const last = merged[merged.length - 1];
            if (loadedRanges[i].start <= last.end + 1) {
                last.end = Math.max(last.end, loadedRanges[i].end);
            } else {
                merged.push(loadedRanges[i]);
            }
        }
        loadedRanges = merged;
    }

    function clearLineSelection() {
        fileContent.querySelectorAll(".line.selected").forEach(el => {
            el.classList.remove("selected");
        });
        selectedRange = null;
        hideAddCommentBar();
        history.replaceState(null, "", window.location.pathname + window.location.search);
    }

    function updateLineSelection(start, end) {
        fileContent.querySelectorAll(".line.selected").forEach(el => {
            el.classList.remove("selected");
        });
        selectedRange = { start: Math.min(start, end), end: Math.max(start, end) };
        for (let i = selectedRange.start; i <= selectedRange.end; i++) {
            const el = fileContent.querySelector(`.line[data-line="${i}"]`);
            if (el) el.classList.add("selected");
        }
        const hash = selectedRange.start === selectedRange.end
            ? `#L${selectedRange.start}`
            : `#L${selectedRange.start}-L${selectedRange.end}`;
        history.replaceState(null, "", hash);

        if (rwMode) {
            showAddCommentBar();
        }
    }

    function handleLineNumberClick(e, lineNum) {
        e.preventDefault();
        if (e.shiftKey && selectedRange) {
            updateLineSelection(selectedRange.start, lineNum);
        } else {
            updateLineSelection(lineNum, lineNum);
        }
    }

    function createLineElement(num, text) {
        const div = document.createElement("div");
        div.className = "line";
        div.dataset.line = num;

        const numSpan = document.createElement("span");
        numSpan.className = "line-number";
        numSpan.textContent = num;
        numSpan.addEventListener("click", (e) => handleLineNumberClick(e, num));

        const textSpan = document.createElement("span");
        textSpan.className = "line-text";
        textSpan.textContent = text;

        div.appendChild(numSpan);
        div.appendChild(textSpan);

        // Apply selected class if this line is in the current selection
        if (selectedRange && num >= selectedRange.start && num <= selectedRange.end) {
            div.classList.add("selected");
        }

        return div;
    }

    function insertLines(data) {
        const lines = data.lines;
        if (!lines || lines.length === 0) return;

        const existingEls = Array.from(fileContent.querySelectorAll(".line"));
        let existingIdx = 0;

        for (const line of lines) {
            while (existingIdx < existingEls.length &&
                   parseInt(existingEls[existingIdx].dataset.line) < line.number) {
                existingIdx++;
            }

            if (existingIdx < existingEls.length &&
                parseInt(existingEls[existingIdx].dataset.line) === line.number) {
                continue;
            }

            const el = createLineElement(line.number, line.text);
            if (existingIdx < existingEls.length) {
                fileContent.insertBefore(el, existingEls[existingIdx]);
            } else {
                fileContent.appendChild(el);
            }
        }
    }

    // --- Gap separators ---

    function createGapSeparator(gapStart, gapEnd) {
        const count = gapEnd - gapStart + 1;
        const div = document.createElement("div");
        div.className = "gap-separator";

        const label = document.createElement("span");
        label.className = "gap-label";
        label.textContent = `\u00b7\u00b7\u00b7 ${count.toLocaleString()} line${count !== 1 ? "s" : ""} not loaded (${gapStart}\u2013${gapEnd}) \u00b7\u00b7\u00b7`;

        const actions = document.createElement("span");
        actions.className = "gap-actions";

        const downBtn = document.createElement("button");
        downBtn.className = "gap-btn";
        downBtn.textContent = "\u2193 load down\u2026";
        downBtn.onclick = (e) => {
            if (e.shiftKey) {
                loadGapChunk(gapStart, gapEnd);
            } else {
                loadGapChunk(gapStart, Math.min(gapEnd, gapStart + GAP_CHUNK - 1));
            }
        };
        downBtn.addEventListener("mouseenter", function(e) {
            if (e.shiftKey) { downBtn.classList.add("gap-btn-rainbow"); downBtn.textContent = "\u2193 load all down"; }
        });
        downBtn.addEventListener("mouseleave", function() {
            downBtn.classList.remove("gap-btn-rainbow"); downBtn.textContent = "\u2193 load down\u2026";
        });
        downBtn.addEventListener("mousemove", function(e) {
            if (e.shiftKey) { downBtn.classList.add("gap-btn-rainbow"); downBtn.textContent = "\u2193 load all down"; }
            else { downBtn.classList.remove("gap-btn-rainbow"); downBtn.textContent = "\u2193 load down\u2026"; }
        });

        const upBtn = document.createElement("button");
        upBtn.className = "gap-btn";
        upBtn.textContent = "\u2191 load up\u2026";
        upBtn.onclick = (e) => {
            if (e.shiftKey) {
                loadGapChunk(gapStart, gapEnd);
            } else {
                loadGapChunk(Math.max(gapStart, gapEnd - GAP_CHUNK + 1), gapEnd);
            }
        };
        upBtn.addEventListener("mouseenter", function(e) {
            if (e.shiftKey) { upBtn.classList.add("gap-btn-rainbow"); upBtn.textContent = "\u2191 load all up"; }
        });
        upBtn.addEventListener("mouseleave", function() {
            upBtn.classList.remove("gap-btn-rainbow"); upBtn.textContent = "\u2191 load up\u2026";
        });
        upBtn.addEventListener("mousemove", function(e) {
            if (e.shiftKey) { upBtn.classList.add("gap-btn-rainbow"); upBtn.textContent = "\u2191 load all up"; }
            else { upBtn.classList.remove("gap-btn-rainbow"); upBtn.textContent = "\u2191 load up\u2026"; }
        });

        actions.appendChild(downBtn);
        actions.appendChild(upBtn);
        div.appendChild(label);
        div.appendChild(actions);
        return div;
    }

    function updateGapSeparators() {
        fileContent.querySelectorAll(".gap-separator").forEach(el => el.remove());

        const lines = Array.from(fileContent.querySelectorAll(".line"));
        if (lines.length === 0 || totalLines === 0) return;

        const firstNum = parseInt(lines[0].dataset.line);
        const lastNum = parseInt(lines[lines.length - 1].dataset.line);

        // Top-of-file gap
        if (firstNum > 1) {
            const sep = createGapSeparator(1, firstNum - 1);
            fileContent.insertBefore(sep, lines[0]);
        }

        // Interior gaps
        for (let i = 1; i < lines.length; i++) {
            const prevNum = parseInt(lines[i - 1].dataset.line);
            const currNum = parseInt(lines[i].dataset.line);
            if (currNum > prevNum + 1) {
                const sep = createGapSeparator(prevNum + 1, currNum - 1);
                fileContent.insertBefore(sep, lines[i]);
            }
        }

        // Bottom-of-file gap
        if (lastNum < totalLines) {
            const sep = createGapSeparator(lastNum + 1, totalLines);
            fileContent.appendChild(sep);
        }
    }

    async function loadGapChunk(start, end) {
        if (loading) return;
        loading = true;
        try {
            const data = await fetchLines(start, end);
            totalLines = data.total_lines;
            insertLines(data);
            mergeRange(start, end);
            updateGapSeparators();
        } finally {
            loading = false;
        }
    }

    // --- End gap separators ---

    // --- Add comment bar ---

    let addCommentBar = null;

    function showAddCommentBar() {
        hideAddCommentBar();
        if (!selectedRange) return;

        addCommentBar = document.createElement("div");
        addCommentBar.className = "add-comment-bar";

        const label = document.createElement("span");
        label.className = "add-comment-label";
        label.textContent = `L${selectedRange.start}-${selectedRange.end}`;

        const btn = document.createElement("button");
        btn.className = "comment-btn-save";
        btn.textContent = "Add Comment";
        btn.onclick = () => showAddCommentForm();

        addCommentBar.appendChild(label);
        addCommentBar.appendChild(btn);
        commentContent.parentElement.appendChild(addCommentBar);
    }

    function hideAddCommentBar() {
        if (addCommentBar) {
            addCommentBar.remove();
            addCommentBar = null;
        }
        // Also remove any inline add form
        const form = document.querySelector(".add-comment-form");
        if (form) form.remove();
    }

    function showAddCommentForm() {
        if (!selectedRange) return;
        hideAddCommentBar();

        const form = document.createElement("div");
        form.className = "add-comment-form";

        const heading = document.createElement("div");
        heading.className = "add-comment-heading";
        heading.textContent = `New comment on L${selectedRange.start}-${selectedRange.end}`;

        const textarea = document.createElement("textarea");
        textarea.className = "comment-edit-area";
        textarea.placeholder = "Write your comment...";
        textarea.rows = 4;

        const actions = document.createElement("div");
        actions.className = "comment-form-actions";

        const saveBtn = document.createElement("button");
        saveBtn.className = "comment-btn-save";
        saveBtn.textContent = "Save";
        saveBtn.onclick = async () => {
            const body = textarea.value.trim();
            if (!body) return;
            const author = promptAuthor();
            saveBtn.disabled = true;
            try {
                const resp = await fetch(`/api/comments/create${fileQuery}`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        range_start: selectedRange.start,
                        range_end: selectedRange.end,
                        body: body,
                        author: author,
                    }),
                });
                if (!resp.ok) {
                    const msg = await resp.text();
                    alert("Error creating comment: " + msg);
                    return;
                }
                await reloadComments();
                clearLineSelection();
            } finally {
                saveBtn.disabled = false;
            }
        };

        const cancelBtn = document.createElement("button");
        cancelBtn.className = "comment-btn-cancel";
        cancelBtn.textContent = "Cancel";
        cancelBtn.onclick = () => {
            form.remove();
            if (selectedRange) showAddCommentBar();
        };

        actions.appendChild(saveBtn);
        actions.appendChild(cancelBtn);
        form.appendChild(heading);
        form.appendChild(textarea);
        form.appendChild(actions);
        commentContent.parentElement.appendChild(form);
        textarea.focus();
    }

    // --- End add comment bar ---

    function highlightCommentLines(comment) {
        fileContent.querySelectorAll(".line.highlighted").forEach(el => {
            el.classList.remove("highlighted");
        });

        if (!comment) return;

        for (let i = comment.range_start; i <= comment.range_end; i++) {
            const el = fileContent.querySelector(`.line[data-line="${i}"]`);
            if (el) el.classList.add("highlighted");
        }
    }

    function renderComments() {
        commentContent.innerHTML = "";

        if (comments.length === 0) {
            commentContent.innerHTML = '<div class="loading">No comments found.</div>';
            return;
        }

        for (const c of comments) {
            const block = document.createElement("div");
            block.className = "comment-block" + (c.drifted ? " drifted" : "");

            const header = document.createElement("div");
            header.className = "comment-header";

            const left = document.createElement("span");
            const idSpan = document.createElement("span");
            idSpan.className = "comment-id";
            idSpan.textContent = c.id;

            const rangeSpan = document.createElement("span");
            rangeSpan.className = "comment-range";
            rangeSpan.textContent = ` L${c.range_start}-${c.range_end}`;
            rangeSpan.onclick = () => {
                clearLineSelection();
                scrollToLine(c.range_start);
            };

            left.appendChild(idSpan);
            left.appendChild(rangeSpan);
            header.appendChild(left);

            const rightGroup = document.createElement("span");
            rightGroup.style.display = "flex";
            rightGroup.style.alignItems = "center";
            rightGroup.style.gap = "6px";

            if (c.drifted) {
                const badge = document.createElement("span");
                badge.className = "drift-badge";
                badge.textContent = "DRIFTED";
                badge.title = "Source content has changed since this comment was created";
                rightGroup.appendChild(badge);
            }

            if (rwMode) {
                const editBtn = document.createElement("button");
                editBtn.className = "comment-action-btn";
                editBtn.textContent = "[EDIT]";
                editBtn.title = "Edit comment";
                editBtn.onclick = (e) => {
                    e.stopPropagation();
                    startEditComment(block, c);
                };

                const deleteBtn = document.createElement("button");
                deleteBtn.className = "comment-action-btn comment-action-btn-delete";
                deleteBtn.textContent = "[DEL]";
                deleteBtn.title = "Delete comment";
                deleteBtn.onclick = (e) => {
                    e.stopPropagation();
                    confirmDeleteComment(c);
                };

                rightGroup.appendChild(editBtn);
                rightGroup.appendChild(deleteBtn);
            }

            header.appendChild(rightGroup);

            const body = document.createElement("div");
            body.className = "comment-body";
            body.textContent = c.body;

            block.appendChild(header);
            block.appendChild(body);

            block.addEventListener("mouseenter", async () => {
                await ensureLinesLoaded(c.range_start, c.range_end);
                highlightCommentLines(c);
            });
            block.addEventListener("mouseleave", () => highlightCommentLines(null));

            commentContent.appendChild(block);
        }
    }

    function startEditComment(block, comment) {
        const bodyEl = block.querySelector(".comment-body");
        if (!bodyEl) return;

        const textarea = document.createElement("textarea");
        textarea.className = "comment-edit-area";
        textarea.value = comment.body;
        textarea.rows = Math.max(3, comment.body.split("\n").length + 1);

        const actions = document.createElement("div");
        actions.className = "comment-form-actions";

        const saveBtn = document.createElement("button");
        saveBtn.className = "comment-btn-save";
        saveBtn.textContent = "Save";
        saveBtn.onclick = async () => {
            const newBody = textarea.value.trim();
            if (!newBody) return;
            saveBtn.disabled = true;
            try {
                const resp = await fetch(`/api/comments/update${fileQuery}`, {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ id: comment.id, body: newBody }),
                });
                if (!resp.ok) {
                    const msg = await resp.text();
                    alert("Error updating comment: " + msg);
                    return;
                }
                await reloadComments();
            } finally {
                saveBtn.disabled = false;
            }
        };

        const cancelBtn = document.createElement("button");
        cancelBtn.className = "comment-btn-cancel";
        cancelBtn.textContent = "Cancel";
        cancelBtn.onclick = () => {
            textarea.replaceWith(bodyEl);
            actions.remove();
        };

        actions.appendChild(saveBtn);
        actions.appendChild(cancelBtn);
        bodyEl.replaceWith(textarea);
        block.appendChild(actions);
        textarea.focus();
    }

    async function confirmDeleteComment(comment) {
        if (!confirm(`Delete comment ${comment.id}?`)) return;

        const fileStr = fileParam ? "&file=" + encodeURIComponent(fileParam) : "";
        const resp = await fetch(`/api/comments/delete?id=${encodeURIComponent(comment.id)}${fileStr}`, {
            method: "DELETE",
        });
        if (!resp.ok) {
            const msg = await resp.text();
            alert("Error deleting comment: " + msg);
            return;
        }
        await reloadComments();
    }

    async function reloadComments() {
        const data = await fetchComments();
        comments = data.comments;
        renderComments();
        // Re-show add bar if selection exists
        if (rwMode && selectedRange) {
            showAddCommentBar();
        }
    }

    async function ensureLinesLoaded(start, end) {
        if (isRangeLoaded(start, end)) return;

        const chunkStart = Math.max(1, start - CHUNK_SIZE);
        const chunkEnd = Math.min(totalLines, end + CHUNK_SIZE);

        if (loading) return;
        loading = true;

        try {
            const data = await fetchLines(chunkStart, chunkEnd);
            totalLines = data.total_lines;
            insertLines(data);
            mergeRange(chunkStart, chunkEnd);
            updateGapSeparators();
        } finally {
            loading = false;
        }
    }

    async function scrollToLine(lineNum) {
        await ensureLinesLoaded(lineNum - 10, lineNum + 50);
        const el = fileContent.querySelector(`.line[data-line="${lineNum}"]`);
        if (el) {
            el.scrollIntoView({ behavior: "smooth", block: "center" });
        }
    }

    async function loadMoreLines() {
        if (loading) return;

        const scrollTop = fileContent.scrollTop;
        const scrollHeight = fileContent.scrollHeight;
        const clientHeight = fileContent.clientHeight;

        // Load more when near bottom
        if (scrollHeight - scrollTop - clientHeight < 200) {
            const lastLine = fileContent.querySelector(".line:last-of-type");
            if (!lastLine) return;
            const lastNum = parseInt(lastLine.dataset.line);
            if (lastNum >= totalLines) return;

            const start = lastNum + 1;
            const end = Math.min(totalLines, start + CHUNK_SIZE - 1);
            await ensureLinesLoaded(start, end);
        }

        // Load more when near top
        if (scrollTop < 200) {
            const firstLine = fileContent.querySelector(".line");
            if (!firstLine) return;
            const firstNum = parseInt(firstLine.dataset.line);
            if (firstNum <= 1) return;

            const end = firstNum - 1;
            const start = Math.max(1, end - CHUNK_SIZE + 1);
            await ensureLinesLoaded(start, end);
        }
    }

    fileContent.addEventListener("scroll", () => {
        requestAnimationFrame(loadMoreLines);
    });

    // Initial load
    async function init() {
        fileContent.innerHTML = '<div class="loading">Loading...</div>';
        commentContent.innerHTML = '<div class="loading">Loading...</div>';

        try {
            const [linesData, commentsData, config] = await Promise.all([
                fetchLines(1, CHUNK_SIZE),
                fetchComments(),
                fetchConfig(),
            ]);

            rwMode = config.rw === true;

            totalLines = linesData.total_lines;
            const displayName = fileParam || commentsData.source_file;
            fileInfo.textContent = `${displayName} — ${totalLines} lines, ${commentsData.comments.length} comments`;

            fileContent.innerHTML = "";
            insertLines(linesData);
            mergeRange(1, Math.min(CHUNK_SIZE, totalLines));

            comments = commentsData.comments;
            renderComments();
            updateGapSeparators();

            // In directory mode, add back link and download button before version
            const header = document.getElementById("header");
            const versionEl = document.getElementById("version");
            if (fileParam) {
                const backLink = document.createElement("a");
                backLink.href = "/";
                backLink.textContent = "\u2190 back to directory";
                backLink.style.cssText = "color:#89b4fa;text-decoration:none;margin-left:auto;font-size:12px;";
                backLink.addEventListener("mouseenter", function() { backLink.style.textDecoration = "underline"; });
                backLink.addEventListener("mouseleave", function() { backLink.style.textDecoration = "none"; });
                header.insertBefore(backLink, versionEl);
            }

            // RW badge
            if (rwMode) {
                const rwBadge = document.createElement("span");
                rwBadge.className = "rw-badge";
                rwBadge.textContent = "READ-WRITE";
                header.insertBefore(rwBadge, versionEl);
            }

            // Set up pane-header download links
            const fileDl = document.getElementById("file-download");
            fileDl.href = `/api/download${fileQuery}`;
            fileDl.textContent = "\u2193 download";
            fileDl.style.display = "inline";

            if (comments.length > 0) {
                const commentsDl = document.getElementById("comments-download");
                commentsDl.href = `/api/download-comments${fileQuery}`;
                commentsDl.textContent = "\u2193 download";
                commentsDl.style.display = "inline";
            }

            versionEl.textContent = config.version || "dev";

            await applyHashSelection();
        } catch (err) {
            fileContent.innerHTML = `<div class="loading">Error: ${err.message}</div>`;
        }
    }

    async function applyHashSelection() {
        const match = location.hash.match(/^#L(\d+)(?:-L(\d+))?$/);
        if (!match) return;
        const start = parseInt(match[1]);
        const end = match[2] ? parseInt(match[2]) : start;
        const lo = Math.min(start, end);
        const hi = Math.max(start, end);
        await ensureLinesLoaded(lo, hi);
        updateLineSelection(lo, hi);
        const el = fileContent.querySelector(`.line[data-line="${lo}"]`);
        if (el) el.scrollIntoView({ behavior: "smooth", block: "center" });
    }

    window.addEventListener("hashchange", () => applyHashSelection());

    init();
})();
