# GAME.EXE port status

Every function defined in `u6-re/src/*.c`, with its Go name in `internal/game`.
"ported" means the Go function follows the C; "TODO" means a stub marked `// TODO(port): <C name>` or not written yet.
Ported: 121. TODO: 355.


## seg_0903.c

| C function | status | Go |
|---|---|---|
| LoadFileGr (C_0903_033A) | TODO |  |
| C_0903_0430 | TODO |  |
| C_0903_04DB | ported | loadTileTables |
| C_0903_070B | TODO |  |
| C_0903_0776 | TODO |  |
| C_0903_07C4 | TODO |  |
| PaletteAnimation (C_0903_0820) | ported | paletteAnimation |
| C_0903_09BB | TODO |  |
| C_0903_0A1E | TODO |  |
| C_0903_0AE6 | TODO |  |
| cfg_parse (C_0903_0B46) | TODO |  |

## seg_0A33.c

| C function | status | Go |
|---|---|---|
| SubKarma (C_0A33_0008) | TODO |  |
| AddKarma (C_0A33_0027) | TODO |  |
| SetTileAnimation (C_0A33_0049) | TODO |  |
| C_0A33_0073 | ported | animateTiles |
| C_0A33_0931 | ported | drawClockFace |
| C_0A33_09CE | ported | drawMap |
| OtherAnimations (C_0A33_0CBC) | ported | OtherAnimations |
| C_0A33_0D06 | ported | updateViewNow |
| TerrainDamage (C_0A33_0D1D) | TODO | terrainDamage |
| C_0A33_0FEB | ported | checkStepOn |
| C_0A33_117A | ported | erupt |
| C_0A33_121A | ported | updateMoongates |
| C_0A33_12F6 | ported | changeWind |
| C_0A33_1355 | ported | passTime |
| RefreshStatus (C_0A33_1AB7) | ported | RefreshStatus |
| SetActive (C_0A33_1B36) | ported | SetActive |
| SetPartyMode (C_0A33_1BB2) | ported | SetPartyMode |
| __GetHex (C_0A33_1C4C) | TODO |  |
| C_0A33_1CB4 | ported | Play |

## seg_0C9C.c

| C function | status | Go |
|---|---|---|
| CON_mouseOffAt (C_0C9C_0007) | TODO |  |
| CON_mouseOn (C_0C9C_0050) | TODO |  |
| MkDirection (C_0C9C_0069) | ported | pointerDirection |
| C_0C9C_01FB | ported | updatePointerShape |
| C_0C9C_0286 | ported | putCharAt |
| C_0C9C_02F1 | ported | scrollUp |
| C_0C9C_0397 | TODO |  |
| C_0C9C_042A | ported | loadGame |
| C_0C9C_089F | ported | saveGame |
| C_0C9C_0BCE | ported | drawTileOver |
| C_0C9C_0C10 | TODO |  |
| C_0C9C_0C7D | TODO |  |
| C_0C9C_0D31 | TODO |  |
| C_0C9C_0E57 | TODO |  |
| CON_prompt (C_0C9C_0E6E) | ported | prompt |
| C_0C9C_0F52 | ported | echo |
| __0C9C_102A | TODO |  |
| mkMouseDir (C_0C9C_1054) | ported | mouseDirection |
| mkMouseSelection (C_0C9C_1107) | ported | mouseSelection |
| do_CMD_91 (C_0C9C_121F) | ported | clickPartyView |
| do_CMD_90 (C_0C9C_12B7) | ported | clickPortraitView |
| do_CMD_92 (C_0C9C_131D) | ported | clickEquipmentView |
| do_CMD_A1 (C_0C9C_15EB) | TODO |  |
| setDefaultCommand (C_0C9C_166D) | ported | setDefaultCommand |
| mkMouseCommand (C_0C9C_1714) | ported | mouseCommand |
| __Incant2Spell (C_0C9C_18D8) | TODO |  |
| C_0C9C_1991 | TODO |  |
| C_0C9C_19D6 | TODO |  |
| C_0C9C_1AE5 | TODO | drawStatusCursor |
| C_0C9C_1D59 | ported | readCommand |
| C_0C9C_2A16 | TODO |  |
| CON_getch (C_0C9C_2A59) | ported | getch |
| CON_gets (C_0C9C_2D3A) | ported | gets |
| CON_createClip (C_0C9C_2E09) | ported | createClip |
| CON_setClip (C_0C9C_2E54) | ported | setClip |
| CON_gotoxy (C_0C9C_2E8F) | ported | gotoxy |
| CON_putch0 (C_0C9C_2EFA) | ported | putch0 |
| C_0C9C_32DE | ported | putchRepeat |
| C_0C9C_32F8 | ported | putsRaw |
| CON_atoi (C_0C9C_331A) | ported | formatNumber |
| CON_putch (C_0C9C_33D3) | ported | putch |
| CON_puts (C_0C9C_3621) | ported | puts |
| CON_printf (C_0C9C_3643) | ported | printf |

## seg_101C.c

| C function | status | Go |
|---|---|---|
| cacheChunk (C_101C_000F) | TODO |  |
| C_101C_00D7 | TODO |  |
| C_101C_0306 | ported | refreshArea |
| C_101C_052D | TODO |  |
| C_101C_054C | ported | enterArea |
| PartyEnter (C_101C_0572) | TODO |  |
| PartyExit (C_101C_077E) | TODO |  |
| PartyTeleport (C_101C_0828) | TODO |  |
| C_101C_089E | TODO |  |
| GateTravel (C_101C_0A3A) | TODO |  |
| C_101C_0ACC | TODO |  |
| getChunkIdxAt (C_101C_0CF7) | ported | chunkAt |
| GetTileAtXYZ (C_101C_0DC9) | ported | GetTileAtXYZ |

## seg_1100.c

| C function | status | Go |
|---|---|---|
| __IsVisible (C_1100_000B) | ported | isVisible |
| __1100_009F | TODO |  |
| __SetVisited (C_1100_00CF) | ported | queue |
| C_1100_0131 | ported | computeLight |
| C_1100_0306 | ported | computeView |

## seg_1184.c

| C function | status | Go |
|---|---|---|
| __1184_000E | TODO |  |
| OBJ_MakeDirFrame (C_1184_0073) | ported | directionFrame |
| OBJ_getHead (C_1184_0153) | ported | objectHead |
| C_1184_02FA | ported | listPosition |
| NextLoc (C_1184_053F) | ported | NextLoc |
| FindLoc (C_1184_075A) | ported | FindLoc |
| C_1184_07A7 | ported | findTypeAt |
| FindInv (C_1184_0840) | ported | FindInv |
| NextInv (C_1184_0899) | ported | NextInv |
| NextArea (C_1184_090D) | ported | NextArea |
| SearchArea (C_1184_09DE) | ported | SearchArea |
| NextInvType (C_1184_0A37) | ported | NextInvType |
| FindInvType (C_1184_0AE2) | ported | FindInvType |
| C_1184_0B3D | ported | livingAt |
| C_1184_0BC1 | ported | updateSlimeFrames |
| C_1184_0CBE | ported | moveSerpent |
| C_1184_10F1 | ported | dragonPart |
| C_1184_116D | ported | moveBigCreature |
| AddObj (C_1184_134C) | TODO |  |
| AddMapObj (C_1184_13A7) | ported | AddMapObj |
| AddInvObj (C_1184_14A8) | ported | AddInvObj |
| AddMonster (C_1184_1540) | ported | AddMonster |
| C_1184_1665 | ported | previousInList |
| C_1184_16D5 | ported | unlinkFromArea |
| DeleteObj (C_1184_17A5) | ported | DeleteObj |
| C_1184_19AA | ported | forgetFarObjects |
| C_1184_1B3A | TODO |  |
| C_1184_1C52 | TODO |  |
| C_1184_1D2E | ported | isHeldBy |
| C_1184_1D60 | ported | lastHeld |
| MoveObj (C_1184_1DAF) | ported | MoveObj |
| InsertObj (C_1184_20DD) | ported | InsertObj |
| QuanType (C_1184_21C7) | TODO |  |
| C_1184_21FD | TODO |  |
| GiveObj (C_1184_2241) | TODO |  |
| TakeObj (C_1184_23F8) | TODO |  |
| TestObj (C_1184_252D) | TODO |  |
| TransferObj (C_1184_257D) | TODO |  |
| C_1184_26E2 | ported | deleteTmpFiles |
| C_1184_2722 | ported | writeBlock |
| C_1184_2888 | ported | unloadObjects |
| C_1184_29C4 | ported | compareListOrder |
| C_1184_2B27 | ported | moveHeldAfterHolders |
| C_1184_2DEF | ported | readBlock |
| C_1184_2ECC | ported | rebuildAreaObjects |
| C_1184_2FB3 | ported | unloadLevel |
| C_1184_30A6 | ported | loadLevel |
| LoadNewRegions (C_1184_3176) | ported | LoadNewRegions |
| C_1184_33CA | ported | commitObjects |
| ShowObject (C_1184_347C) | ported | showObject |
| C_1184_35EA | ported | showBigObject |
| ShowObjects (C_1184_36C2) | ported | ShowObjects |
| C_1184_3B1D | ported | resetObjectLists |
| C_1184_3B7D | ported | linkCharacters |
| GetTileString (C_1184_3C1B) | TODO |  |
| GetObjectString (C_1184_3C9C) | TODO | objectName |

## seg_155D.c

| C function | status | Go |
|---|---|---|
| C_155D_000C | ported | partyView |
| STAT_refreshComMode (C_155D_01BC) | TODO |  |
| C_155D_028A | TODO |  |
| STAT_GetEquipSlot (C_155D_0516) | TODO |  |
| C_155D_063A | TODO |  |
| GetWeight (C_155D_0661) | TODO |  |
| Encumbrance (C_155D_0707) | TODO |  |
| C_155D_0748 | TODO |  |
| C_155D_07E0 | TODO |  |
| C_155D_08F4 | TODO |  |
| C_155D_0CB6 | TODO |  |
| C_155D_0CF5 | TODO |  |
| C_155D_1022 | TODO |  |
| C_155D_1065 | TODO |  |
| C_155D_120A | TODO |  |
| C_155D_123E | TODO |  |
| C_155D_1267 | TODO |  |
| C_155D_130E | TODO |  |
| C_155D_13F4 | TODO |  |
| __155D_141B | TODO |  |
| C_155D_144B | TODO |  |
| C_155D_1666 | TODO |  |
| C_155D_16E7 | TODO |  |
| C_155D_1738 | TODO |  |

## seg_16E1.c

| C function | status | Go |
|---|---|---|
| TALK_initTalk (C_16E1_0007) | TODO |  |
| TALK_talkTo (C_16E1_015C) | TODO |  |

## seg_1703.c

| C function | status | Go |
|---|---|---|
| C_1703_00A0 | TODO |  |
| C_1703_0116 | TODO |  |
| C_1703_0153 | TODO |  |
| C_1703_028B | TODO |  |
| JoinParty (C_1703_02B6) | TODO |  |
| C_1703_04C6 | TODO |  |
| C_1703_1494 | TODO |  |
| parse_statement (C_1703_1C45) | TODO |  |
| C_1703_1D01 | TODO |  |
| TalkDriver (C_1703_1E23) | TODO |  |

## seg_1944.c

| C function | status | Go |
|---|---|---|
| C_1944_0054 | TODO |  |
| C_1944_0085 | TODO |  |
| C_1944_01EC | TODO |  |
| C_1944_0232 | TODO |  |
| C_1944_0278 | TODO |  |
| C_1944_02BE | TODO |  |
| SPELL_RefreshList (C_1944_0321) | TODO |  |
| SPELL_DrawBG (C_1944_05B1) | TODO |  |
| C_1944_06CA | TODO |  |
| TakeReagents (C_1944_06EF) | TODO |  |
| SPELL_RefreshPage (C_1944_0773) | TODO |  |
| C_1944_0A43 | TODO |  |
| C_1944_0AA9 | TODO |  |
| C_1944_0AD0 | TODO |  |
| C_1944_0B32 | TODO |  |
| C_1944_0B85 | TODO |  |
| C_1944_0BF4 | TODO |  |
| C_1944_0CA3 | TODO |  |
| C_1944_0D46 | TODO |  |
| C_1944_0DCF | TODO |  |
| C_1944_0F02 | TODO |  |
| C_1944_1038 | TODO |  |
| C_1944_114D | TODO |  |
| C_1944_11BB | TODO |  |
| C_1944_1232 | TODO |  |
| C_1944_12FC | TODO |  |
| C_1944_1A42 | TODO |  |
| C_1944_1C51 | TODO |  |
| __1944_1DAB | TODO |  |
| C_1944_1DF4 | TODO |  |
| C_1944_22F4 | TODO |  |
| C_1944_23F2 | TODO |  |
| C_1944_256A | TODO |  |
| C_1944_27AF | TODO |  |
| C_1944_29AA | TODO |  |
| C_1944_2A15 | TODO |  |
| C_1944_2B9A | TODO |  |
| C_1944_2BFD | TODO |  |
| C_1944_2DA7 | TODO |  |
| C_1944_305A | TODO |  |
| C_1944_310C | TODO |  |
| C_1944_3175 | TODO |  |
| C_1944_34CF | TODO |  |
| C_1944_35EC | TODO |  |
| C_1944_3750 | TODO |  |
| C_1944_39A1 | TODO |  |
| C_1944_3B2D | TODO |  |
| C_1944_3C30 | TODO |  |
| C_1944_3D0F | TODO |  |
| C_1944_3DB8 | TODO |  |
| C_1944_3F6C | TODO |  |
| C_1944_4260 | TODO |  |
| C_1944_42AC | TODO |  |
| C_1944_435B | TODO |  |
| C_1944_43A3 | TODO |  |
| C_1944_43D5 | TODO |  |
| C_1944_4926 | TODO |  |
| C_1944_4C2F | TODO |  |

## seg_1E0F.c

| C function | status | Go |
|---|---|---|
| C_1E0F_000F | ported | passable |
| IsOutOfArea (C_1E0F_056E) | ported | isOutOfArea |
| C_1E0F_05D8 | ported | blockedAt |
| C_1E0F_0664 | ported | turnTo |
| SubMov (C_1E0F_0F54) | ported | subMove |
| C_1E0F_0FA9 | ported | draggedUnderTurn |
| C_1E0F_1056 | ported | alreadyNear |
| MoveFollowers (C_1E0F_1193) | ported | MoveFollowers |
| Board (C_1E0F_14BE) | TODO |  |
| Unboard (C_1E0F_1745) | TODO |  |
| C_1E0F_184D | ported | enterGates |
| C_1E0F_1B0E | ported | advance |
| C_1E0F_2001 | ported | openDoorUnder |
| C_1E0F_2125 | TODO |  |
| C_1E0F_2184 | TODO |  |
| __AtDestination (C_1E0F_2276) | TODO | atDestination |
| C_1E0F_25F9 | TODO |  |
| C_1E0F_291C | TODO | leavePath |
| C_1E0F_2A74 | TODO |  |
| __ResetPath (C_1E0F_2C9D) | TODO |  |
| C_1E0F_2D37 | TODO |  |
| C_1E0F_31C7 | TODO |  |
| SubTerrainMov (C_1E0F_31F7) | ported | subTerrainMove |
| TryStraightMove (C_1E0F_32D6) | ported | TryStraightMove |
| C_1E0F_33C4 | TODO |  |
| __TryDiagMove (C_1E0F_3470) | ported | tryDiagonalMove |
| TryMoveTo (C_1E0F_35A7) | ported | TryMoveTo |
| C_1E0F_37DB | TODO |  |
| __DoOnPath (C_1E0F_387D) | TODO |  |
| C_1E0F_39B3 | TODO |  |
| C_1E0F_3B60 | TODO |  |
| C_1E0F_3D8D | TODO |  |
| C_1E0F_3E08 | TODO |  |
| __NewPathIndex (C_1E0F_3E27) | TODO |  |
| C_1E0F_3E6A | TODO | npcTurn |
| C_1E0F_4265 | TODO |  |
| __ComputeResistance (C_1E0F_436A) | TODO |  |
| C_1E0F_464A | TODO | teleportNPCs |
| C_1E0F_4746 | TODO | placeScheduledNPCs |
| C_1E0F_4AFA | TODO |  |
| C_1E0F_4B6A | TODO | updateLeaders |
| C_1E0F_4E0A | ported | runTurns |
| C_1E0F_512A | ported | startTurns |
| C_1E0F_5165 | ported | followSchedules |

## seg_2337.c

| C function | status | Go |
|---|---|---|
| COMBAT_getCathesus (C_2337_000B) | ported | combatRange |
| COMBAT_stopCharm (C_2337_005E) | TODO | stopCharm |
| COMBAT_begin (C_2337_00C8) | TODO | beginCombat |
| COMBAT_breakOff (C_2337_01BC) | TODO |  |
| COMBAT_TryTeleport (C_2337_02DC) | TODO |  |
| GetStr (C_2337_03E7) | TODO |  |
| GetDex (C_2337_0411) | TODO |  |
| GetInt (C_2337_0458) | TODO |  |
| COMBAT_TestIntel (C_2337_0482) | TODO |  |
| ComputeLevel (__2337_04BE) | TODO |  |
| GetLevel (C_2337_04E1) | TODO |  |
| AddXP (C_2337_04F4) | TODO |  |
| MaxHP (C_2337_0529) | ported | maxHitPoints |
| MaxMagic (C_2337_0559) | ported | maxMagic |
| SetTypeUnconscious (C_2337_05B4) | TODO |  |
| COMBAT_canSee (C_2337_0837) | TODO |  |
| C_2337_08F1 | TODO | objectAtForSelection |
| GetArmorPoints (C_2337_0983) | TODO |  |
| __GetProtection (C_2337_09BF) | TODO |  |
| __GetWeaponRange (C_2337_0A14) | TODO |  |
| C_2337_0A4E | TODO |  |
| COMBAT_Missile (C_2337_0B3F) | TODO |  |
| C_2337_0B99 | TODO |  |
| GetDamagePoints (C_2337_10E4) | TODO |  |
| CloneObject (C_2337_115F) | TODO |  |
| COMBAT_showHit (C_2337_121F) | TODO |  |
| LooseHP (C_2337_12F5) | TODO | loseHitPoints |
| __AttackTest (C_2337_18CF) | TODO |  |
| COMBAT_displaySeverity (C_2337_198F) | TODO |  |
| C_2337_1A99 | TODO |  |
| C_2337_1B0E | TODO |  |
| C_2337_1C1E | TODO |  |
| __DoHit (C_2337_1D20) | TODO |  |
| COMBAT_getHead (C_2337_205C) | TODO |  |
| __TryHit (C_2337_214D) | TODO |  |
| C_2337_27C5 | TODO |  |
| C_2337_2875 | TODO |  |
| COMBAT_attack (C_2337_2AA9) | TODO |  |
| COMBAT_range (C_2337_2BEE) | TODO |  |
| COMBAT_pickOpponent (C_2337_2D6E) | TODO |  |
| COMBAT_AI_Assault (C_2337_2F3E) | TODO |  |
| C_2337_310E | TODO |  |
| COMBAT_AI_Front (C_2337_3464) | TODO |  |
| COMBAT_AI_Rear (C_2337_34C9) | TODO |  |
| COMBAT_AI_Flank (C_2337_37F2) | TODO |  |
| COMBAT_AI_Bersek (C_2337_3C3A) | TODO |  |
| COMBAT_AI_Retreat (C_2337_3ED2) | TODO |  |
| COMBAT_AI_Like (C_2337_407F) | TODO |  |
| COMBAT_AI_Unfriendly (C_2337_41B7) | TODO |  |
| COMBAT_AI_Immobile (C_2337_42EE) | TODO |  |
| __TryGrowVine (C_2337_43D2) | TODO |  |
| COMBAT_AI_Tangle (C_2337_44F6) | TODO |  |

## seg_27a1.c

| C function | status | Go |
|---|---|---|
| C_27A1_0008 | TODO |  |
| C_27A1_0028 | TODO |  |
| C_27A1_0051 | TODO |  |
| C_27A1_0082 | TODO |  |
| C_27A1_00A9 | TODO |  |
| C_27A1_012C | TODO |  |
| IsOkToGet (C_27A1_0153) | TODO |  |
| SetOkToGet (C_27A1_019E) | TODO |  |
| C_27A1_01DE | TODO |  |
| C_27A1_0205 | TODO |  |
| C_27A1_02D9 | TODO |  |
| C_27A1_061E | TODO |  |
| C_27A1_066D | TODO |  |
| C_27A1_06A2 | TODO |  |
| C_27A1_06D7 | TODO |  |
| C_27A1_078F | TODO |  |
| C_27A1_0841 | TODO |  |
| C_27A1_08D3 | TODO |  |
| C_27A1_0919 | TODO |  |
| C_27A1_09A1 | TODO |  |
| C_27A1_0C67 | TODO |  |
| C_27A1_1330 | TODO |  |
| C_27A1_13BE | TODO |  |
| C_27A1_14DA | TODO |  |
| C_27A1_18F5 | TODO |  |
| C_27A1_1DAB | TODO |  |
| C_27A1_1E69 | TODO |  |
| C_27A1_1E8B | TODO |  |
| C_27A1_28A3 | TODO |  |
| C_27A1_2A44 | TODO |  |
| C_27A1_2BBC | TODO |  |
| C_27A1_2D34 | TODO |  |
| C_27A1_2D61 | TODO |  |
| C_27A1_2D8E | TODO |  |
| C_27A1_2FD1 | TODO |  |
| C_27A1_319F | TODO |  |
| C_27A1_31F6 | TODO |  |
| C_27A1_32FA | TODO |  |
| C_27A1_335A | TODO |  |
| C_27A1_338D | TODO |  |
| C_27A1_3425 | TODO |  |
| C_27A1_3537 | TODO |  |
| __27A1_3617 | TODO |  |
| __27A1_363C | TODO |  |
| C_27A1_3661 | TODO |  |
| C_27A1_36E7 | TODO |  |
| C_27A1_376D | TODO |  |
| C_27A1_37D3 | TODO |  |
| C_27A1_3832 | TODO |  |
| C_27A1_3A35 | TODO |  |
| C_27A1_3B59 | TODO |  |
| AfterExplosion (C_27A1_3BE1) | TODO |  |
| Detonate (C_27A1_3D2B) | TODO | detonate |
| __SearchTypeAt (C_27A1_3E21) | TODO |  |
| C_27A1_3E59 | TODO |  |
| C_27A1_3E9A | TODO |  |
| C_27A1_3ED1 | TODO |  |
| C_27A1_3EF6 | TODO |  |
| C_27A1_3F47 | TODO |  |
| C_27A1_433D | TODO |  |
| C_27A1_4479 | TODO |  |
| C_27A1_4672 | TODO |  |
| C_27A1_47E0 | TODO |  |
| C_27A1_47F3 | TODO |  |
| C_27A1_49C3 | TODO |  |
| C_27A1_4B0B | TODO |  |
| C_27A1_4B98 | TODO |  |
| C_27A1_4D46 | TODO |  |
| C_27A1_4E6F | TODO |  |
| C_27A1_4E9B | TODO |  |
| C_27A1_4FC6 | TODO |  |
| C_27A1_51C4 | TODO |  |
| C_27A1_5289 | TODO |  |
| C_27A1_5431 | TODO |  |
| C_27A1_5503 | TODO |  |
| C_27A1_55F0 | TODO |  |
| C_27A1_5789 | TODO |  |
| C_27A1_5935 | TODO |  |
| C_27A1_5A97 | TODO |  |
| C_27A1_5BCF | TODO |  |
| C_27A1_5DF2 | TODO |  |
| C_27A1_5F43 | TODO |  |
| C_27A1_5FAC | TODO |  |
| C_27A1_60BD | TODO |  |
| C_27A1_60D6 | TODO |  |
| C_27A1_60F5 | TODO |  |
| C_27A1_6179 | TODO |  |

## seg_2E2D.c

| C function | status | Go |
|---|---|---|
| __2E2D_0003 | TODO |  |
| GetMonsterClass (C_2E2D_004B) | ported | monsterClass |
| mkRandom (C_2E2D_0088) | TODO |  |
| C_2E2D_00BE | TODO |  |
| EGG_generate (C_2E2D_0499) | TODO |  |
| EGG_hatches (C_2E2D_0760) | TODO |  |
| EGG_hatchArea (C_2E2D_0DFE) | TODO | hatchEggs |

## seg_2F1A.c

| C function | status | Go |
|---|---|---|
| MUS_0008 | TODO |  |
| MUS_0065 | TODO |  |
| MUS_043B | TODO |  |
| MUS_051E | TODO |  |
| MUS_0525 | TODO |  |
| MUS_07F6 | TODO |  |
| MUS_0851 | TODO |  |
| MUS_091A | TODO |  |
| MUS_09A8 | TODO |  |
| MUS_0A3A | TODO |  |

## seg_2FC1.c

| C function | status | Go |
|---|---|---|
| C_2FC1_000A | TODO |  |
| MagicalHalo (C_2FC1_00B0) | TODO |  |
| C_2FC1_017F | TODO |  |
| C_2FC1_0212 | TODO |  |
| C_2FC1_032B | TODO |  |
| C_2FC1_131E | TODO |  |
| C_2FC1_13B0 | TODO |  |
| ShakeScreen (C_2FC1_18BD) | TODO | shakeScreen |
| C_2FC1_193C | TODO |  |
| C_2FC1_19A1 | TODO |  |
| C_2FC1_19C5 | ported | drawSky |
| C_2FC1_1BCE | TODO |  |
| C_2FC1_1C19 | TODO |  |
| LoadConversation (C_2FC1_1D40) | TODO |  |
| C_2FC1_1EAF | TODO |  |
| peer_tile (C_2FC1_1F14) | TODO |  |
| C_2FC1_1FD6 | TODO |  |

## seg_3200.c

| C function | status | Go |
|---|---|---|
| IsEdible (C_3200_000E) | TODO |  |
| TryEatExclusive (C_3200_0035) | TODO |  |
| TryEat (C_3200_0094) | TODO |  |
| C_3200_00D3 | TODO |  |
| C_3200_0160 | TODO |  |
| C_3200_0226 | TODO |  |
| C_3200_02C4 | TODO |  |
| HasNightEncounter (C_3200_0309) | TODO |  |
| C_3200_0467 | TODO |  |
| C_3200_0514 | TODO |  |
| C_3200_055D | TODO |  |

## seg_32C3.c

| C function | status | Go |
|---|---|---|
| DISK_0005 | TODO |  |
| DISK_0008 | TODO |  |
| DISK_0053 | TODO |  |
| DISK_confirm (DISK_0097) | TODO |  |
| DISK_0202 | TODO |  |
| DISK_full (DISK_0211) | TODO |  |
| DISK_init (DISK_0266) | TODO |  |
