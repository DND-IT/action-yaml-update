import { parseInputs, InputError } from "../src/inputs.js";

describe("parseInputs", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    // Clear all INPUT_ vars
    Object.keys(process.env).forEach((key) => {
      if (key.startsWith("INPUT_")) {
        delete process.env[key];
      }
    });
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  test("parses key mode inputs", () => {
    process.env.INPUT_FILES = "file1.yaml\nfile2.yaml";
    process.env.INPUT_MODE = "key";
    process.env.INPUT_KEYS = "app.version\napp.tag";
    process.env.INPUT_VALUES = "v1.0.0\nlatest";

    const inputs = parseInputs();

    expect(inputs.files).toEqual(["file1.yaml", "file2.yaml"]);
    expect(inputs.mode).toBe("key");
    expect(inputs.keys).toEqual(["app.version", "app.tag"]);
    expect(inputs.values).toEqual(["v1.0.0", "latest"]);
  });

  test("parses image mode inputs", () => {
    process.env.INPUT_FILES = "values.yaml";
    process.env.INPUT_MODE = "image";
    process.env.INPUT_IMAGE_NAME = "webapp";
    process.env.INPUT_IMAGE_TAG = "v2.0.0";

    const inputs = parseInputs();

    expect(inputs.mode).toBe("image");
    expect(inputs.imageName).toBe("webapp");
    expect(inputs.imageTag).toBe("v2.0.0");
  });

  test("throws on missing files", () => {
    expect(() => parseInputs()).toThrow(InputError);
    expect(() => parseInputs()).toThrow("'files' input is required");
  });

  test("throws on invalid mode", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "invalid";

    expect(() => parseInputs()).toThrow("Invalid mode");
  });

  test("throws on missing keys for key mode", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "key";

    expect(() => parseInputs()).toThrow("'keys' input is required");
  });

  test("throws on missing values for key mode", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "key";
    process.env.INPUT_KEYS = "app.version";

    expect(() => parseInputs()).toThrow("'values' input is required");
  });

  test("throws on mismatched keys/values count", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "key";
    process.env.INPUT_KEYS = "a\nb";
    process.env.INPUT_VALUES = "1";

    expect(() => parseInputs()).toThrow("must match");
  });

  test("throws on missing image_name for image mode", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "image";

    expect(() => parseInputs()).toThrow("'image_name' input is required");
  });

  test("throws on missing image_tag for image mode", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "image";
    process.env.INPUT_IMAGE_NAME = "webapp";

    expect(() => parseInputs()).toThrow("'image_tag' input is required");
  });

  test("parses boolean values", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "key";
    process.env.INPUT_KEYS = "x";
    process.env.INPUT_VALUES = "y";
    process.env.INPUT_CREATE_PR = "false";
    process.env.INPUT_DRY_RUN = "true";
    process.env.INPUT_AUTO_MERGE = "yes";

    const inputs = parseInputs();

    expect(inputs.createPr).toBe(false);
    expect(inputs.dryRun).toBe(true);
    expect(inputs.autoMerge).toBe(true);
  });

  test("parses comma-separated lists", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "key";
    process.env.INPUT_KEYS = "x";
    process.env.INPUT_VALUES = "y";
    process.env.INPUT_PR_LABELS = "bug,enhancement";
    process.env.INPUT_PR_REVIEWERS = "user1,user2";

    const inputs = parseInputs();

    expect(inputs.prLabels).toEqual(["bug", "enhancement"]);
    expect(inputs.prReviewers).toEqual(["user1", "user2"]);
  });

  test("uses default values", () => {
    process.env.INPUT_FILES = "test.yaml";
    process.env.INPUT_MODE = "key";
    process.env.INPUT_KEYS = "x";
    process.env.INPUT_VALUES = "y";

    const inputs = parseInputs();

    expect(inputs.createPr).toBe(true);
    expect(inputs.prTitle).toBe("chore: update YAML values");
    expect(inputs.commitMessage).toBe("chore: update YAML values");
    expect(inputs.mergeMethod).toBe("SQUASH");
    expect(inputs.gitUserName).toBe("github-actions[bot]");
  });
});
