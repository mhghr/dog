import { ApiError } from "./errors";

export interface ToastMessage {
  title: string;
  description?: string;
}

const MESSAGES: Record<
  string,
  { title: { en: string; fa: string }; description: { en: string; fa: string } }
> = {
  network_error: {
    title: { en: "Network error", fa: "خطای شبکه" },
    description: {
      en: "Could not reach the server. Please try again.",
      fa: "اتصال به سرور برقرار نشد. دوباره تلاش کنید.",
    },
  },
  not_found: {
    title: { en: "Not found", fa: "یافت نشد" },
    description: {
      en: "The requested item does not exist.",
      fa: "مورد مورد نظر وجود ندارد.",
    },
  },
  duplicate: {
    title: { en: "Already exists", fa: "مقدار تکراری" },
    description: {
      en: "A resource with the same value already exists.",
      fa: "منبعی با همین مقدار از قبل وجود دارد.",
    },
  },
  invalid_id: {
    title: { en: "Invalid ID", fa: "شناسه نامعتبر" },
    description: {
      en: "The provided ID is not valid.",
      fa: "شناسه ارسال شده معتبر نیست.",
    },
  },
  invalid_json: {
    title: { en: "Invalid request", fa: "درخواست نامعتبر" },
    description: {
      en: "The request body could not be read.",
      fa: "بدنه درخواست قابل خواندن نیست.",
    },
  },
  invalid_range: {
    title: { en: "Invalid time range", fa: "بازه زمانی نامعتبر" },
    description: {
      en: "The requested time range is invalid.",
      fa: "بازه زمانی درخواستی نامعتبر است.",
    },
  },
  internal_error: {
    title: { en: "Something went wrong", fa: "خطای سرور" },
    description: {
      en: "An unexpected error occurred. Please try again.",
      fa: "خطای غیرمنتظره‌ای رخ داد. دوباره تلاش کنید.",
    },
  },
};

function firstFieldError(fields?: Record<string, string[]>): string | undefined {
  if (!fields) return undefined;
  for (const key of Object.keys(fields)) {
    const messages = fields[key];
    if (Array.isArray(messages) && messages.length > 0) return messages[0];
  }
  return undefined;
}

/**
 * Convert an unknown thrown value into a localized, user-facing toast message.
 * Prefer this over `err.message` directly so backend codes map to friendly,
 * consistent copy and validation fields surface their first message.
 */
export function apiErrorMessage(err: unknown, isFa: boolean): ToastMessage {
  const lang = isFa ? "fa" : "en";

  if (err instanceof ApiError) {
    const known = MESSAGES[err.code ?? ""];
    if (known) {
      return {
        title: known.title[lang],
        description: known.description[lang],
      };
    }

    const fieldMessage = firstFieldError(err.fields);
    if (fieldMessage) {
      return { title: isFa ? "خطا" : "Error", description: fieldMessage };
    }

    if (err.message && err.code !== "internal_error") {
      return {
        title: isFa ? "خطا" : "Error",
        description: err.message,
      };
    }

    return {
      title: isFa ? "خطا" : "Error",
      description: isFa ? "مشکلی پیش آمد. دوباره تلاش کنید." : "Something went wrong. Please try again.",
    };
  }

  return {
    title: isFa ? "خطا" : "Error",
    description: isFa ? "مشکلی پیش آمد." : "Something went wrong.",
  };
}
