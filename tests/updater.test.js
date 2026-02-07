import {
  loadYaml,
  dumpYaml,
  updateKeys,
  updateImageTags,
  diffYaml,
} from "../src/updater.js";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const FIXTURES_DIR = join(__dirname, "fixtures");

describe("updateKeys", () => {
  test("simple key update", () => {
    const doc = loadYaml("app:\n  version: v1.0.0\n");
    const changes = updateKeys(doc, ["app.version"], ["v2.0.0"]);

    expect(changes).toHaveLength(1);
    expect(changes[0].old).toBe("v1.0.0");
    expect(changes[0].new).toBe("v2.0.0");
    expect(doc.getIn(["app", "version"])).toBe("v2.0.0");
  });

  test("nested key update", () => {
    const doc = loadYaml("a:\n  b:\n    c: old\n");
    const changes = updateKeys(doc, ["a.b.c"], ["new"]);

    expect(changes).toHaveLength(1);
    expect(doc.getIn(["a", "b", "c"])).toBe("new");
  });

  test("multiple keys", () => {
    const doc = loadYaml("x: 1\ny: 2\n");
    const changes = updateKeys(doc, ["x", "y"], ["10", "20"]);

    expect(changes).toHaveLength(2);
    expect(doc.getIn(["x"])).toBe(10);
    expect(doc.getIn(["y"])).toBe(20);
  });

  test("list index", () => {
    const doc = loadYaml("items:\n  - name: a\n    value: old\n");
    const changes = updateKeys(doc, ["items.0.value"], ["new"]);

    expect(changes).toHaveLength(1);
    expect(doc.getIn(["items", 0, "value"])).toBe("new");
  });

  test("no change when same value", () => {
    const doc = loadYaml("key: same\n");
    const changes = updateKeys(doc, ["key"], ["same"]);

    expect(changes).toHaveLength(0);
  });

  test("key not found throws", () => {
    const doc = loadYaml("key: value\n");
    expect(() => updateKeys(doc, ["missing.path"], ["val"])).toThrow(
      "not found",
    );
  });
});

describe("type coercion", () => {
  test("int stays int", () => {
    const doc = loadYaml("replicas: 3\n");
    updateKeys(doc, ["replicas"], ["5"]);

    expect(doc.getIn(["replicas"])).toBe(5);
    expect(typeof doc.getIn(["replicas"])).toBe("number");
  });

  test("bool stays bool", () => {
    const doc = loadYaml("enabled: true\n");
    updateKeys(doc, ["enabled"], ["false"]);

    expect(doc.getIn(["enabled"])).toBe(false);
  });

  test("float stays float", () => {
    const doc = loadYaml("ratio: 1.5\n");
    updateKeys(doc, ["ratio"], ["2.5"]);

    expect(doc.getIn(["ratio"])).toBe(2.5);
  });

  test("string stays string", () => {
    const doc = loadYaml("name: hello\n");
    updateKeys(doc, ["name"], ["world"]);

    expect(doc.getIn(["name"])).toBe("world");
  });

  test("int with non-numeric becomes string", () => {
    const doc = loadYaml("port: 8080\n");
    updateKeys(doc, ["port"], ["not-a-number"]);

    expect(doc.getIn(["port"])).toBe("not-a-number");
  });
});

describe("updateImageTags", () => {
  test("helm style match", () => {
    const doc = loadYaml(
      "image:\n  repository: ghcr.io/myorg/webapp\n  tag: v1.0.0\n",
    );
    const changes = updateImageTags(doc, "webapp", "v2.0.0");

    expect(changes).toHaveLength(1);
    expect(changes[0].old).toBe("v1.0.0");
    expect(dumpYaml(doc)).toContain("tag: v2.0.0");
  });

  test("helm style no match", () => {
    const doc = loadYaml(
      "image:\n  repository: ghcr.io/myorg/other\n  tag: v1.0.0\n",
    );
    const changes = updateImageTags(doc, "webapp", "v2.0.0");

    expect(changes).toHaveLength(0);
    expect(dumpYaml(doc)).toContain("tag: v1.0.0");
  });

  test("kustomize style match", () => {
    const doc = loadYaml(
      "images:\n  - name: ghcr.io/myorg/webapp\n    newTag: v1.0.0\n",
    );
    const changes = updateImageTags(doc, "webapp", "v2.0.0");

    expect(changes).toHaveLength(1);
    expect(dumpYaml(doc)).toContain("newTag: v2.0.0");
  });

  test("multiple matches", () => {
    const content = readFileSync(
      join(FIXTURES_DIR, "multi_image.yaml"),
      "utf8",
    );
    const doc = loadYaml(content);
    const changes = updateImageTags(doc, "api", "v5.0.0");

    expect(changes).toHaveLength(2);
  });

  test("same tag no change", () => {
    const doc = loadYaml(
      "image:\n  repository: ghcr.io/myorg/webapp\n  tag: v1.0.0\n",
    );
    const changes = updateImageTags(doc, "webapp", "v1.0.0");

    expect(changes).toHaveLength(0);
  });

  test("exact name match", () => {
    const doc = loadYaml("image:\n  repository: webapp\n  tag: v1.0.0\n");
    const changes = updateImageTags(doc, "webapp", "v2.0.0");

    expect(changes).toHaveLength(1);
  });
});

describe("format preservation", () => {
  test("comments preserved after key update", () => {
    const content = readFileSync(
      join(FIXTURES_DIR, "comments_preserved.yaml"),
      "utf8",
    );
    const doc = loadYaml(content);
    updateKeys(doc, ["app.version"], ["2.0.0"]);
    const result = dumpYaml(doc);

    expect(result).toContain("# Top-level comment");
    expect(result).toContain("# This is the version");
    expect(result).toContain("# inline comment");
  });

  test("indentation preserved - 2 space", () => {
    const content = `app:
  ports:
    - name: http
      port: 8080
  image:
    repository: myapp
    tag: v1.0.0
`;
    const doc = loadYaml(content);
    updateKeys(doc, ["app.image.tag"], ["v2.0.0"]);
    const result = dumpYaml(doc);

    expect(result).toContain("  ports:");
    expect(result).toContain("    - name: http");
  });

  test("indentation preserved - 4 space", () => {
    const content = `app:
    ports:
        - name: http
          port: 8080
    image:
        repository: myapp
        tag: v1.0.0
`;
    const doc = loadYaml(content);
    updateKeys(doc, ["app.image.tag"], ["v2.0.0"]);
    const result = dumpYaml(doc);

    expect(result).toContain("    ports:");
    expect(result).toContain("        - name: http");
  });
});

describe("diffYaml", () => {
  test("diff shows changes", () => {
    const content = "app:\n  version: v1.0.0\n";
    const doc = loadYaml(content);
    updateKeys(doc, ["app.version"], ["v2.0.0"]);
    const result = diffYaml("test.yaml", content, doc);

    expect(result).toContain("-  version: v1.0.0");
    expect(result).toContain("+  version: v2.0.0");
  });

  test("diff empty when no changes", () => {
    const content = "app:\n  version: v1.0.0\n";
    const doc = loadYaml(content);
    const result = diffYaml("test.yaml", content, doc);

    expect(result).toBe("");
  });
});
