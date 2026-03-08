import i18n from "i18next";

// Zod v4 error map for i18next translation.
// Maps Zod issue codes to translated messages using the "zod" namespace.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function zodErrorMap(issue: any): { message: string } {
  const t = i18n.t.bind(i18n);

  let message: string;

  switch (issue.code) {
    case "invalid_type":
      if (issue.input === undefined) {
        message = t("zod:errors.invalid_type_received_undefined");
      } else if (issue.input === null) {
        message = t("zod:errors.invalid_type_received_null");
      } else {
        message = t("zod:errors.invalid_type", {
          expected: issue.expected,
          received: typeof issue.input,
        });
      }
      break;

    case "too_small":
      if (issue.origin === "string") {
        message = issue.minimum === 1
          ? t("zod:errors.invalid_type_received_undefined")
          : t("zod:errors.too_small.string.inclusive", { minimum: issue.minimum });
      } else if (issue.origin === "number") {
        message = t("zod:errors.too_small.number.inclusive", { minimum: issue.minimum });
      } else if (issue.origin === "array") {
        message = t("zod:errors.too_small.array.inclusive", { minimum: issue.minimum });
      } else {
        message = t("zod:errors.too_small.string.inclusive", { minimum: issue.minimum });
      }
      break;

    case "too_big":
      if (issue.origin === "string") {
        message = t("zod:errors.too_big.string.inclusive", { maximum: issue.maximum });
      } else if (issue.origin === "number") {
        message = t("zod:errors.too_big.number.inclusive", { maximum: issue.maximum });
      } else if (issue.origin === "array") {
        message = t("zod:errors.too_big.array.inclusive", { maximum: issue.maximum });
      } else {
        message = t("zod:errors.too_big.string.inclusive", { maximum: issue.maximum });
      }
      break;

    case "invalid_format":
      message = t("zod:errors.invalid_string.regex");
      break;

    case "invalid_value":
      message = t("zod:errors.invalid_enum_value", {
        options: (issue as unknown as { values: string[] }).values?.join(", ") ?? "",
        received: String(issue.input),
      });
      break;

    default:
      // Let Zod use its default message
      issue.continue = true;
      message = "";
      break;
  }

  return { message };
}
