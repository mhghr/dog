import type { Icon, IconProps } from "@phosphor-icons/react";
import {
  Broadcast as BroadcastBase,
  Browser as BrowserBase,
  CalendarCheck as CalendarCheckBase,
  Clock as ClockBase,
  EnvelopeSimple as EnvelopeSimpleBase,
  Gauge as GaugeBase,
  GearSix as GearSixBase,
  Globe as GlobeBase,
  List as ListBase,
  MagnifyingGlass as MagnifyingGlassBase,
  MapPin as MapPinBase,
  Moon as MoonBase,
  PlugsConnected as PlugsConnectedBase,
  Pulse as PulseBase,
  ShieldCheck as ShieldCheckBase,
  ShieldWarning as ShieldWarningBase,
  SignOut as SignOutBase,
  SquaresFour as SquaresFourBase,
  Sun as SunBase,
  Timer as TimerBase,
  Tray as TrayBase,
  TreeStructure as TreeStructureBase,
  UserCircle as UserCircleBase,
  Warning as WarningBase,
} from "@phosphor-icons/react/dist/ssr";

// Design-system icon source of truth. The SSR entrypoint is context-free, so
// these components are safe in both server and client components; the filled
// weight is baked in here instead of a client-only IconContext.
function filled(Base: Icon): Icon {
  function FilledIcon(props: IconProps) {
    return <Base weight="fill" {...props} />;
  }
  return FilledIcon as Icon;
}

export const Broadcast = filled(BroadcastBase);
export const Browser = filled(BrowserBase);
export const CalendarCheck = filled(CalendarCheckBase);
export const Clock = filled(ClockBase);
export const EnvelopeSimple = filled(EnvelopeSimpleBase);
export const Gauge = filled(GaugeBase);
export const GearSix = filled(GearSixBase);
export const Globe = filled(GlobeBase);
export const List = filled(ListBase);
export const MagnifyingGlass = filled(MagnifyingGlassBase);
export const MapPin = filled(MapPinBase);
export const Moon = filled(MoonBase);
export const PlugsConnected = filled(PlugsConnectedBase);
export const Pulse = filled(PulseBase);
export const ShieldCheck = filled(ShieldCheckBase);
export const ShieldWarning = filled(ShieldWarningBase);
export const SignOut = filled(SignOutBase);
export const SquaresFour = filled(SquaresFourBase);
export const Sun = filled(SunBase);
export const Timer = filled(TimerBase);
export const Tray = filled(TrayBase);
export const TreeStructure = filled(TreeStructureBase);
export const UserCircle = filled(UserCircleBase);
export const Warning = filled(WarningBase);

export type { Icon as AppIcon } from "@phosphor-icons/react";
