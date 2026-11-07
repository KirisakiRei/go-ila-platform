import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * cn menggabungkan kelas Tailwind dan menyelesaikan konflik
 * (mis. px-2 lalu px-4) dengan hasil akhir yang deterministik.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}
