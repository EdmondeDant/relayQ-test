import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const source = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), "../GroupsView.vue"),
  "utf8",
);

describe("GroupsView Leonardo", () => {
  it("offers Leonardo once for creation and once for filtering", () => {
    expect(source.match(/\{ value: "leonardo", label: "Leonardo" \}/g)).toHaveLength(2);
    expect(source).toContain('platform: "anthropic" as GroupPlatform');
    expect(source).toContain("...createForm");
    expect(source).toContain("platform: (filters.platform as GroupPlatform) || undefined");
  });

  it("hydrates the Leonardo platform without exposing unrelated controls", () => {
    expect(source).toContain("editForm.platform = group.platform");
    expect(source).toContain('createForm.platform === "openai"');
    expect(source).toContain("createForm.platform === 'antigravity'");
    expect(source).toContain("createForm.platform === 'anthropic'");
    expect(source).not.toContain("createForm.platform === 'leonardo' ||");
    expect(source).not.toContain("editForm.platform === 'leonardo' ||");
    expect(source).toContain('if (newVal === "leonardo")');
    expect(source).toContain('if (group.platform === "leonardo")');
    expect(source).toContain("createForm.mcp_xml_inject = false");
    expect(source).toContain("editForm.mcp_xml_inject = false");
  });

  it("renders Leonardo through the translated platform label", () => {
    expect(source).toContain('t("admin.groups.platforms." + value)');
    expect(source).toContain('t("admin.groups.platforms." + group.platform)');
  });
});
